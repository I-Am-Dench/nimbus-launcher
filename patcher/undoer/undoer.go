package undoer

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/client"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CreateChangesTable    = "CREATE TABLE IF NOT EXISTS changes (packed BOOLEAN NOT NULL, root TEXT NOT NULL, path TEXT NOT NULL, state INTEGER, uncompressed_size INTEGER DEFAULT 0, uncompressed_hash BLOB, compressed_size INTEGER DEFAULT 0, compressed_hash BLOB, compressed BOOLEAN DEFAULT 0, data BLOB)"
	InsertAddedChange     = "INSERT INTO changes (packed, root, path, state) VALUES (?, ?, ?, ?)"
	InsertReplacedChange  = "INSERT INTO changes VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	SelectPackedChanges   = "SELECT path, state, uncompressed_size, uncompressed_hash, compressed_size, compressed_hash, compressed, data FROM changes WHERE packed = 1 AND root = ?"
	SelectUnpackedChanges = "SELECT path, state, data FROM changes WHERE packed = 0 AND root = ?"
	DeletePackedChanges   = "DELETE FROM changes WHERE packed = ? AND root = ? AND state <> 0"
	DeleteUnpackedChanges = "DELETE FROM changes WHERE packed = ? AND root = ?"
)

type State int

const (
	Added = State(iota)
	Replaced
)

// Undoer cumulatively tracks which resources have been added and/or replaced
// by a patcher. Undoers are not guaranteed to track the order in which changes
// have been made, only what changes have been made.
type Undoer interface {
	Track(path string, archive *client.Archive) error
	Undo(archive *client.Archive) error
}

type sqliteUndoer struct {
	*sql.DB

	root string
}

func NewSqlite(name, root string) (Undoer, error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, fmt.Errorf("undoer: sqlite: %w", err)
	}

	if _, err := db.Exec(CreateChangesTable); err != nil {
		return nil, fmt.Errorf("undoer: sqlite: %w", err)
	}

	return &sqliteUndoer{db, filepath.Clean(root)}, nil
}

func (u *sqliteUndoer) added(packed bool, path string) error {
	if _, err := u.Exec(InsertAddedChange, packed, u.root, filepath.Clean(path), Added); err != nil {
		return fmt.Errorf("undoer: sqlite: added: %w", err)
	}
	return nil
}

func (u *sqliteUndoer) replaced(packed bool, path string, info archive.Info, compressed bool, data []byte) error {
	if _, err := u.Exec(InsertReplacedChange, packed, u.root, filepath.Clean(path), Replaced, info.UncompressedSize, info.UncompressedChecksum, info.CompressedSize, info.CompressedChecksum, compressed, data); err != nil {
		return fmt.Errorf("undoer: sqlite: replaced: %w", err)
	}
	return nil
}

func (u *sqliteUndoer) trackPacked(path string, archive *client.Archive) error {
	pack, err := archive.FindPack(path)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: %w", err)
	}

	record, ok := pack.Search(path)
	if !ok {
		return u.added(true, path)
	}

	section, _, err := record.Section()
	if err != nil {
		return fmt.Errorf("undoer: sqlite: %w", err)
	}

	data, err := io.ReadAll(section)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: %w", err)
	}

	return u.replaced(true, path, record.Info, record.IsCompressed, data)
}

func (u *sqliteUndoer) trackUnpacked(path string) error {
	data, err := os.ReadFile(filepath.Join(u.root, path))
	if err == nil {
		return u.replaced(false, path, archive.Info{}, false, data)
	} else {
		return u.added(false, path)
	}
}

func (u *sqliteUndoer) Track(path string, archive *client.Archive) error {
	if archive != nil {
		return u.trackPacked(path, archive)
	} else {
		return u.trackUnpacked(path)
	}
}

func (u *sqliteUndoer) undoPacked(a *client.Archive) error {
	rows, err := u.Query(SelectPackedChanges, u.root)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %w", err)
	}

	errs := []error{}
	for rows.Next() {
		var (
			path       string
			state      State
			info       archive.Info
			compressed bool
			data       sql.RawBytes
		)
		if err := rows.Scan(&path, &state, &info.UncompressedSize, &info.UncompressedChecksum, &info.CompressedSize, &info.CompressedChecksum, &compressed, &data); err != nil {
			return fmt.Errorf("undoer: sqlite: undo: %w", err)
		}

		switch state {
		case Added:
			// Since the pack implementation does not have a way to
			// remove entries, we'll leave it alone, but continue
			// tracking it after the undo.
		case Replaced:
			pack, err := a.FindPack(path)
			if err != nil {
				return fmt.Errorf("undoer: sqlite: undo: %w", err)
			}

			if err := pack.Store(path, info, compressed, bytes.NewReader(data)); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, fmt.Errorf("illegal state: State(%d)", state))
		}
	}

	if _, err := u.Exec(DeletePackedChanges, true, u.root); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %w", err)
	}

	return nil
}

func (u *sqliteUndoer) undoUnpacked() error {
	rows, err := u.Query(SelectUnpackedChanges, u.root)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %w", err)
	}

	errs := []error{}
	for rows.Next() {
		var (
			path  string
			state State
			data  sql.RawBytes
		)
		if err := rows.Scan(&path, &state, &data); err != nil {
			return fmt.Errorf("undoer: sqlite: undo: %w", err)
		}

		switch state {
		case Added:
			os.Remove(filepath.Join(u.root, path))
		case Replaced:
			if data == nil {
				data = []byte{}
			}

			if err := os.WriteFile(filepath.Join(u.root, path), data, 0755); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, fmt.Errorf("illegal state: State(%d)", state))
		}
	}

	if _, err := u.Exec(DeleteUnpackedChanges, false, u.root); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %w", err)
	}

	return nil
}

func (u *sqliteUndoer) Undo(archive *client.Archive) error {
	if archive != nil {
		return u.undoPacked(archive)
	} else {
		return u.undoUnpacked()
	}
}
