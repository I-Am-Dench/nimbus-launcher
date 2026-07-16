package tracker_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
)

const BaseClientDir = "testdata"

func catalogEntries() archive.CatalogEntries {
	return archive.CatalogEntries{
		"client/pack/pack0.pk": []archive.CatalogEntry{
			{Path: "client/file0.txt"},
			{Path: "client/file1.txt"},
			{Path: "client/file2.txt"},
		},
		"client/pack/pack1.pk": []archive.CatalogEntry{
			{Path: "client/data0/file0.txt"},
			{Path: "client/data0/file1.txt"},
		},
	}
}

func flattenEntries(catalogEntries archive.CatalogEntries) iter.Seq[archive.CatalogEntry] {
	return func(yield func(archive.CatalogEntry) bool) {
		for _, entries := range catalogEntries {
			for _, entry := range entries {
				if !yield(entry) {
					return
				}
			}
		}
	}
}

func generateData() []byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	b := make([]byte, rand.IntN(128)+64)
	for i := range b {
		b[i] = chars[rand.IntN(len(chars))]
	}
	return b
}

type Client struct {
	CachePath string
	BasePath  string
	BaseFiles map[string][]byte

	Tracker tracker.Tracker
}

func (c Client) Files(includePack bool) iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		for path, expectedData := range c.BaseFiles {
			if !includePack && (strings.HasSuffix(path, ".pk") || strings.HasSuffix(path, ".pki")) {
				continue
			}

			if !yield(path, expectedData) {
				return
			}
		}
	}
}

func (c *Client) Reset() error {
	if err := os.RemoveAll(c.BasePath); err != nil {
		return fmt.Errorf("reset: %v", err)
	}

	for name, data := range c.BaseFiles {
		path := filepath.Join(c.BasePath, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("reset: write data: %v", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("reset: write data: %v", err)
		}
	}

	if err := os.RemoveAll(c.CachePath); err != nil {
		return fmt.Errorf("reset: remove cache: %v", err)
	}

	if err := os.MkdirAll(c.CachePath, 0755); err != nil {
		return fmt.Errorf("reset: %v", err)
	}

	tkr, err := tracker.New(c.CachePath, c.BasePath)
	if err != nil {
		return fmt.Errorf("reset: %v", err)
	}
	c.Tracker = tkr

	return nil
}

func (c Client) writeUnpacked(name string, r io.Reader) error {
	file, err := os.Create(filepath.Join(c.BasePath, name))
	if err != nil {
		return fmt.Errorf("write unpacked: %s: %v", name, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("write unpacked: %s: %v", name, err)
	}
	return nil
}

func (c Client) writePacked(name string, r io.Reader, ar *archive.Archive) error {
	record, ok := ar.Catalog().Search(name)
	if !ok {
		return c.writeUnpacked(name, r)
	}

	compressedData := bytes.Buffer{}

	info, err := archive.CalculateInfoFromReader(r, &compressedData)
	if err != nil {
		return fmt.Errorf("write packed: %s: %v", name, err)
	}

	data := r
	if record.IsCompressed {
		data = &compressedData
	}

	if err := ar.Store(name, info, record.IsCompressed, data); err != nil {
		return fmt.Errorf("write packed: %s: %v", name, err)
	}
	return nil
}

func (c Client) Write(name string, r io.Reader, ar *archive.Archive) error {
	if err := c.Tracker.Track(name, ar); err != nil {
		return fmt.Errorf("write: %s: %v", name, err)
	}

	if ar == nil {
		return c.writeUnpacked(name, r)
	} else {
		return c.writePacked(name, r, ar)
	}
}

func (c Client) WriteRandom(name string, ar *archive.Archive) error {
	return c.Write(name, bytes.NewReader(generateData()), ar)
}

func (c Client) WriteCatalog(name string, entries archive.CatalogEntries) error {
	temp, err := os.CreateTemp(c.CachePath, "catalog*.pki")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	catalog, err := archive.NewCatalog(temp)
	if err != nil {
		return err
	}
	defer catalog.Close()

	if err := catalog.Store(entries); err != nil {
		return err
	}

	if err := catalog.Flush(); err != nil {
		return err
	}

	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := c.Write(name, temp, nil); err != nil {
		return err
	}

	return nil
}

func (c Client) Check(t *testing.T) {
	// TODO: check "additions" file

	for path, expectedData := range c.Files(true) {
		t.Log("client: CHECKING FILE:", path)

		actualData, err := os.ReadFile(filepath.Join(c.BasePath, path))
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expectedData, actualData) {
			t.Errorf("%s: expected %q but got %q", path, string(expectedData), string(actualData))
		}
	}
}

func (c *Client) Test(f func(*testing.T, *Client)) func(*testing.T) {
	return func(t *testing.T) {
		t.Log("client: RESETTING CLIENT")
		if err := c.Reset(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := c.Tracker.Close(); err != nil {
				t.Error(err)
			}
		}()

		t.Log("client: RUNNING MODIFICATIONS")
		f(t, c)

		t.Log("client: UNDOING CLIENT")
		if err := c.Tracker.Undo(context.Background()); err != nil {
			t.Error(err)
			return
		}

		c.Check(t)
	}
}

func collectFiles(dir string) (map[string][]byte, error) {
	m := make(map[string][]byte)
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		m[rel] = data

		return nil
	}); err != nil {
		return nil, err
	}
	return m, nil
}

func setup(t *testing.T) (client Client, teardown func(), err error) {
	tempDir, err := os.MkdirTemp(".", "cache-*.tmp")
	if err != nil {
		return client, nil, fmt.Errorf("setup: %v", err)
	}

	teardown = func() {
		if os.Getenv("KEEP_TESTDATA") != "1" {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Log(err)
			}
		}
	}

	files, err := collectFiles(BaseClientDir)
	if err != nil {
		teardown()
		return client, nil, fmt.Errorf("setup: %v", err)
	}

	client = Client{
		CachePath: filepath.Join(tempDir, "cache"),
		BasePath:  filepath.Join(tempDir, "base"),
		BaseFiles: files,
	}

	if err := client.Reset(); err != nil {
		teardown()
		return client, nil, fmt.Errorf("setup: %v", err)
	}

	return client, teardown, nil
}

// Only update unpacked files
func unpackedSimple(t *testing.T, client *Client) {
	for path := range client.Files(false) {
		t.Log("Updating", path)
		if err := client.WriteRandom(path, nil); err != nil {
			t.Error(err)
		}
	}
}

// Only add unpacked files
func unpackedAdditions(t *testing.T, client *Client) {
	for i := range 10 {
		path := filepath.Join("client", fmt.Sprint("added", i, ".txt"))

		t.Log("Adding", path)
		if err := client.WriteRandom(path, nil); err != nil {
			t.Error(err)
		}
	}
}

// Update cataloged and uncataloged files
func packedSimple(t *testing.T, client *Client) {
	ar, err := archive.Open(client.BasePath, filepath.Join(client.BasePath, "primary.pki"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := ar.Close(); err != nil {
			t.Error(err)
		}
	}()

	for path := range client.Files(false) {
		t.Log("Updating", path)
		if err := client.WriteRandom(path, ar); err != nil {
			t.Error(err)
		}
	}
}

func packedAdditions(t *testing.T, client *Client) {
	entries := catalogEntries()
	for packName := range entries {
		basePackName, _ := strings.CutSuffix(filepath.Base(packName), ".pk")
		for i := range 5 {
			path := filepath.Join("client", fmt.Sprint(basePackName, "_addition", i, ".txt"))
			t.Logf("Adding %s to catalog", path)
			entries[packName] = append(entries[packName], archive.CatalogEntry{Path: path})
		}
	}

	t.Logf("Updating catalog")
	if err := client.WriteCatalog("primary.pki", entries); err != nil {
		t.Fatal(err)
	}

	ar, err := archive.Open(client.BasePath, filepath.Join(client.BasePath, "primary.pki"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := ar.Close(); err != nil {
			t.Error(err)
		}
	}()

	for entry := range flattenEntries(entries) {
		t.Log("Updating", entry.Path)
		if err := client.WriteRandom(entry.Path, ar); err != nil {
			t.Error(err)
		}
	}
}

func many(client *Client) func(t *testing.T) {
	return func(t *testing.T) {
		if err := client.Reset(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := client.Tracker.Close(); err != nil {
				t.Error(err)
			}
		}()

		ar, err := archive.Open(client.BasePath, filepath.Join(client.BasePath, "primary.pki"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := ar.Close(); err != nil {
				t.Error(err)
			}
		}()

		for i := range 15 {
			for path := range client.Files(false) {
				t.Logf("%d: Updating %s", i+1, path)
				if err := client.WriteRandom(path, ar); err != nil {
					t.Error(err)
				}
			}

			t.Logf("Undoing client")
			if err := client.Tracker.Undo(context.Background()); err != nil {
				t.Fatal(err)
			}
		}

		client.Check(t)
	}
}

func TestTracker(t *testing.T) {
	client, teardown, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	t.Run("unpackedSimple", client.Test(unpackedSimple))
	t.Run("unpackedAdditions", client.Test(unpackedAdditions))
	t.Run("packedSimple", client.Test(packedSimple))
	t.Run("packedAdditions", client.Test(packedAdditions))
	t.Run("many", many(&client))
}
