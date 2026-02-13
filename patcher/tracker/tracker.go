package tracker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/I-Am-Dench/goverbuild/archive"
)

const (
	AdditionsName = "additions"

	NewInstallName = ".nimbus-new"
)

type State int

const (
	StateNormal = State(iota)
	StateNew
)

type Tracker interface {
	Track(path string, archive *archive.Archive) error
	Undo(ctx context.Context) error
	Close() error

	GetState() State
	SetState(State)
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

type fsTracker struct {
	cacheDir, clientCacheDir, root string

	state     State
	additions Additions
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

	state := StateNormal
	if _, err := os.Stat(filepath.Join(cacheDir, NewInstallName)); err == nil {
		state = StateNew
	}

	return &fsTracker{cacheDir, clientCacheDir, root, state, additions}, nil
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

func (t fsTracker) trackUnpacked(path string) error {
	if _, ok := t.additions[filepath.Clean(path)]; ok {
		return nil // resource is an addition, no need to cache
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

func (t fsTracker) trackPacked(path string, archive *archive.Archive) error {
	// don't need to track the individual files,
	// just the packs themselves
	record, ok := archive.Catalog().Search(path)
	if ok {
		return t.trackUnpacked(record.PackName)
	} else {
		return t.trackUnpacked(path)
	}
}

func (t fsTracker) Track(path string, archive *archive.Archive) error {
	if t.state == StateNew {
		return nil // Don't track for new clients
	}

	if archive == nil {
		return t.trackUnpacked(path)
	} else {
		return t.trackPacked(path, archive)
	}
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

		// Allows users to manually add entries to "additions" files.
		//
		// This shouldn't get entered under normal use, since resources
		// won't get cached anyway.
		if _, ok := t.additions[filepath.Clean(path)]; ok {
			return nil
		}

		clientFile, err := os.Create(filepath.Join(t.root, rel))
		if err != nil {
			return err
		}
		defer clientFile.Close()

		cachedFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer cachedFile.Close()

		_, err = io.Copy(clientFile, cachedFile)
		return err
	}
}

func (t fsTracker) Undo(ctx context.Context) error {
	if err := filepath.WalkDir(t.clientCacheDir, t.undo(ctx)); err != nil {
		return fmt.Errorf("undo: %w", err)
	}
	return nil
}

func (t fsTracker) Close() error {
	return t.writeAdditions()
}

func (t fsTracker) GetState() State {
	return t.state
}

func (t *fsTracker) SetState(state State) {
	t.state = state

	newFilePath := filepath.Join(t.cacheDir, NewInstallName)
	switch state {
	case StateNew:
		os.Create(newFilePath)
	default:
		os.Remove(newFilePath)
	}
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
