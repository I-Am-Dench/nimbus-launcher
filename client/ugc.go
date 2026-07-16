package client

import (
	"bufio"
	"bytes"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type UgcManifest map[string]time.Time

func ReadUgcManifest(name string) (UgcManifest, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read ugc manifest: %w", err)
	}
	defer file.Close()

	manifest := make(UgcManifest)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		path, lastModifiedStr, _ := bytes.Cut(scanner.Bytes(), []byte(","))

		lastModified, err := strconv.ParseInt(string(lastModifiedStr), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("read ugc manifest: %v", err)
		}

		manifest[string(path)] = time.Unix(lastModified, 0)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ugc manifest: %v", err)
	}
	return manifest, nil
}

func WriteUgcManifest(name string, manifest UgcManifest) error {
	keys := slices.Collect(maps.Keys(manifest))
	slices.SortFunc(keys, func(a, b string) int { return strings.Compare(a, b) })

	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("write ugc manifest: %v", err)
	}
	defer file.Close()

	for _, path := range keys {
		if _, err := fmt.Fprint(file, path, ",", manifest[path].Unix(), "\n"); err != nil {
			return fmt.Errorf("write ugc manifest: %v", err)
		}
	}

	return nil
}
