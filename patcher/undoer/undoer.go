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
	_ "github.com/mattn/go-sqlite3"
)

const (
	createChangesTable = `
CREATE TABLE IF NOT EXISTS changes (
	packed BOOLEAN NOT NULL,
	root TEXT NOT NULL,
	path TEXT NOT NULL,
	state INTEGER,
	uncompressed_size INTEGER NOT NULL DEFAULT 0,
	uncompressed_hash BLOB,
	compressed_size INTEGER NOT NULL DEFAULT 0,
	compressed_hash BLOB,
	compressed BOOLEAN DEFAULT 0,
	data BLOB
)`

	insertAddedChange     = `INSERT INTO changes (packed, root, path, state) VALUES (?, ?, ?, ?)`
	insertReplacedChange  = `INSERT INTO changes VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectPackedChanges   = `SELECT path, state, uncompressed_size, uncompressed_hash, compressed_size, compressed_hash, compressed, data FROM changes WHERE packed <> 0 AND root = ?`
	selectUnpackedChanges = `SELECT path, state, data FROM changes WHERE packed = 0 AND root = ?`
	deletePackedChanges   = `DELETE FROM changes WHERE packed <> 0 AND root = ? AND state <> 0`
	deleteUnpackedChanges = `DELETE FROM changes WHERE packed = 0 AND root = ?`

	countPackedAddedChanges = `SELECT COUNT(*) FROM changes WHERE packed <> 0 AND root = ? AND path = ?`
)

type State int

const (
	Added = State(iota)
	Replaced
)

type Undoer interface {
	Track(path string, archive *archive.Archive) error
	Undo(archive *archive.Archive) error
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

	if _, err := db.Exec(createChangesTable); err != nil {
		return nil, fmt.Errorf("undoer: sqlite: %w", err)
	}

	return &sqliteUndoer{db, root}, nil
}

func (u *sqliteUndoer) added(packed bool, path string) error {
	if _, err := u.Exec(insertAddedChange, packed, u.root, filepath.Clean(path), Added); err != nil {
		return fmt.Errorf("undoer: sqlite: added: %v", err)
	}
	return nil
}

func (u *sqliteUndoer) replaced(packed bool, path string, info archive.Info, compressed bool, data []byte) error {
	if _, err := u.Exec(insertReplacedChange,
		packed,
		u.root,
		filepath.Clean(path),
		Replaced,
		info.UncompressedSize,
		info.UncompressedChecksum,
		info.CompressedSize,
		info.CompressedChecksum,
		compressed,
		data,
	); err != nil {
		return fmt.Errorf("undoer: sqlite: replaced: %v", err)
	}
	return nil
}

func (u *sqliteUndoer) trackPacked(path string, arch *archive.Archive) error {
	record, err := arch.Load(path)
	if errors.Is(err, archive.ErrNotPacked) {
		return u.added(true, path)
	}

	if err != nil {
		return fmt.Errorf("undoer: sqlite: %v", err)
	}

	var count int
	row := u.QueryRow(countPackedAddedChanges, u.root, filepath.Clean(path))
	row.Scan(&count)

	if count > 0 {
		return nil // Packed resource has already been marked as "Added"
	}

	section, err := record.Section()
	if err != nil {
		return fmt.Errorf("undoer: sqlite: %v", err)
	}

	data, err := io.ReadAll(section)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: %v", err)
	}

	return u.replaced(true, path, record.Info, record.IsCompressed, data)
}

func (u *sqliteUndoer) trackUnpacked(path string) error {
	data, err := os.ReadFile(filepath.Join(u.root, filepath.Clean(path)))
	if err == nil {
		return u.replaced(false, path, archive.Info{}, false, data)
	} else {
		return u.added(false, path)
	}
}

func (u *sqliteUndoer) Track(path string, archive *archive.Archive) error {
	if archive != nil {
		return u.trackPacked(path, archive)
	} else {
		return u.trackUnpacked(path)
	}
}

func (u *sqliteUndoer) undoPacked(arch *archive.Archive) error {
	rows, err := u.Query(selectPackedChanges, u.root)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %v", err)
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
		if err := rows.Scan(
			&path,
			&state,
			&info.UncompressedSize,
			&info.UncompressedChecksum,
			&info.CompressedSize,
			&info.CompressedChecksum,
			&compressed,
			&data,
		); err != nil {
			return fmt.Errorf("undoer: sqlite: undo: %v", err)
		}

		switch state {
		case Added:
			// Since the pack implementation does not have a way to
			// remove entries, we'll leave it alone, but continue
			// tracking it after the undo.
		case Replaced:
			if err := arch.Store(path, info, compressed, bytes.NewReader(data)); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, fmt.Errorf("illegal state: State(%d)", state))
		}
	}

	if _, err := u.Exec(deletePackedChanges, u.root); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %v", err)
	}

	return nil
}

func (u *sqliteUndoer) undoUnpacked() error {
	rows, err := u.Query(selectUnpackedChanges, u.root)
	if err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %v", err)
	}

	errs := []error{}
	for rows.Next() {
		var (
			path  string
			state State
			data  sql.RawBytes
		)
		if err := rows.Scan(&path, &state, &data); err != nil {
			return fmt.Errorf("undoer: sqlite: undo: %v", err)
		}

		switch state {
		case Added:
			os.Remove(filepath.Join(u.root, path))
		case Replaced:
			if data == nil {
				data = []byte{}
			}

			if err := os.WriteFile(filepath.Join(u.root, filepath.Clean(path)), data, 0664); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, err)
		}
	}

	if _, err := u.Exec(deleteUnpackedChanges, u.root); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("undoer: sqlite: undo: %v", err)
	}

	return nil
}

func (u *sqliteUndoer) Undo(archive *archive.Archive) error {
	if archive != nil {
		return u.undoPacked(archive)
	} else {
		return u.undoUnpacked()
	}
}
