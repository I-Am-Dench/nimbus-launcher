package tracker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/I-Am-Dench/goverbuild/archive"
)

const (
	AdditionsName = "additions"
	StateName     = "state"
)

type State struct {
	Forward bool   `json:"foward"`
	Profile string `json:"profile"`
}

type Tracker interface {
	Track(path string, archive *archive.Archive) error
	Undo(ctx context.Context) error
	Close() error

	State() *State
}

type Additions map[string]struct{}

func readAdditions(name string) (Additions, error) {
	data, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return Additions{}, nil
	} else if err != nil {
		return nil, err
	}

	additions := Additions{}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		additions[filepath.Clean(scanner.Text())] = struct{}{}
	}

	return additions, nil
}

func readState(name string) (State, error) {
	data, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	} else if err != nil {
		return State{}, err
	}

	state := State{}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

type fsTracker struct {
	cacheDir, clientCacheDir, root string

	additions Additions
	state     State
}

func New(cacheDir, root string) (Tracker, error) {
	clientCacheDir := filepath.Join(cacheDir, "installation")
	if err := os.MkdirAll(clientCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("tracker: %w", err)
	}

	additions, err := readAdditions(filepath.Join(cacheDir, AdditionsName))
	if err != nil {
		return nil, fmt.Errorf("tracker: %w", err)
	}

	state, err := readState(filepath.Join(cacheDir, StateName))
	if err != nil {
		return nil, fmt.Errorf("tracker: %w", err)
	}

	return &fsTracker{cacheDir, clientCacheDir, root, additions, state}, nil
}

func (t fsTracker) writeAdditions() error {
	file, err := os.Create(filepath.Join(t.cacheDir, AdditionsName))
	if err != nil {
		return err
	}
	defer file.Close()

	for addition := range t.additions {
		fmt.Fprintln(file, addition)
	}

	return nil
}

func (t fsTracker) writeState() error {
	data, err := json.Marshal(t.state)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(t.cacheDir, StateName), data, 0664); err != nil {
		return err
	}

	return nil
}

func (t fsTracker) trackUnpacked(path string) error {
	if _, ok := t.additions[filepath.Clean(path)]; ok {
		return nil // resource is an addition; no need to cache
	}

	clientFile, err := os.Open(filepath.Join(t.root, path))
	if errors.Is(err, os.ErrNotExist) {
		t.additions[filepath.Clean(path)] = struct{}{}
		return nil // resource is not in base client, thus is an addition
	} else if err != nil {
		return fmt.Errorf("track: %v", err)
	}
	defer clientFile.Close()

	cachedPath := filepath.Join(t.clientCacheDir, path)
	if _, err := os.Stat(cachedPath); err == nil {
		return nil // resource is already cached
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("track: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		return fmt.Errorf("track: %v", err)
	}

	cachedFile, err := os.Create(cachedPath)
	if err != nil {
		return fmt.Errorf("track: %v", err)
	}
	defer cachedFile.Close()

	if _, err := io.Copy(cachedFile, clientFile); err != nil {
		return fmt.Errorf("track: %v", err)
	}

	return nil
}

func (t fsTracker) trackPacked(path string, ar *archive.Archive) error {
	// don't need to track the individual files,
	// just the packs themselves
	record, ok := ar.Catalog().Search(path)
	if ok {
		return t.trackUnpacked(record.PackName)
	} else {
		return t.trackUnpacked(path)
	}
}

func (t fsTracker) Track(path string, archive *archive.Archive) error {
	if t.state.Forward {
		return nil // Don't track for foward update clients
	}

	if archive == nil {
		return t.trackUnpacked(path)
	} else {
		return t.trackPacked(path, archive)
	}
}

func (f fsTracker) shouldCopy(clientPath string, cachedStat fs.FileInfo) bool {
	clientStat, err := os.Stat(clientPath)
	if err != nil {
		return true
	}

	return !cachedStat.ModTime().Equal(clientStat.ModTime()) || cachedStat.Size() != clientStat.Size()
}

func (t fsTracker) undo(ctx context.Context) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(t.clientCacheDir, path)
		if err != nil {
			return err
		}

		// Ignoring additions takes precendence over
		// replacing base files.
		if _, ok := t.additions[filepath.Clean(path)]; ok {
			return nil
		}

		cachedStat, err := d.Info()
		if err != nil {
			return err
		}

		clientPath := filepath.Join(t.root, rel)
		if !t.shouldCopy(clientPath, cachedStat) {
			return nil
		}

		clientFile, err := os.Create(clientPath)
		if err != nil {
			return err
		}
		defer clientFile.Close()

		cachedFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer cachedFile.Close()

		if _, err := io.Copy(clientFile, cachedFile); err != nil {
			return err
		}
		os.Chtimes(clientPath, time.Time{}, cachedStat.ModTime())

		return nil
	}
}

func (t fsTracker) Undo(ctx context.Context) error {
	if err := filepath.WalkDir(t.clientCacheDir, t.undo(ctx)); err != nil {
		return fmt.Errorf("undo: %w", err)
	}
	return nil
}

func (t fsTracker) Close() error {
	return errors.Join(t.writeAdditions(), t.writeState())
}

func (t *fsTracker) State() *State {
	return &t.state
}

func HashFilepath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	if runtime.GOOS != "linux" {
		abs = strings.ToUpper(abs) // case insensitive
	}

	hash := md5.New()
	hash.Write([]byte(abs))

	return hex.EncodeToString(hash.Sum(nil)), nil
}
