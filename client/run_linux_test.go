//go:build linux
// +build linux

package client_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/I-Am-Dench/nimbus-launcher/client"
)

type protonVersion struct {
	Name        string
	Version     int
	VersionName string
}

func (version *protonVersion) WriteVersion(dir string) error {
	data := []byte(fmt.Sprint(version.Version, " ", version.VersionName))

	if err := os.WriteFile(filepath.Join(dir, version.Name, "version"), data, 0755); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	return nil
}

func (version *protonVersion) TestRecentVersion(dir string) error {
	proton, err := client.FindRecentProtonVersion(dir)
	if err != nil {
		return err
	}

	if dir := filepath.Base(filepath.Dir(proton)); dir != version.Name {
		return fmt.Errorf("expected \"%s\", but got \"%s\"", version.Name, dir)
	}

	return nil
}

func TestFindRecentProtonVersion(t *testing.T) {
	temp, err := os.MkdirTemp(".", "client_test_*")
	if err != nil {
		t.Fatalf("test find proton: %v", err)
	}
	defer os.RemoveAll(temp)

	versions := []protonVersion{
		{"Proton 1.0", 100, "1.0"},
		{"Proton 2.0", 128, "2.0"},
		{"Proton 3.0", 339, "3.0"},
		{"Proton 4.0 (Beta)", 450, "4.0-b"},
		{"Proton 4.0", 507, "4.0"},
	}

	for _, version := range versions {
		if err := os.MkdirAll(filepath.Join(temp, version.Name), 0755); err != nil {
			t.Fatalf("test find proton: %v", err)
		}

		if err := version.WriteVersion(temp); err != nil {
			t.Fatalf("test find proton: %v", err)
		}
	}

	t.Log("TEST: recent version")
	if err := versions[4].TestRecentVersion(temp); err != nil {
		t.Errorf("test find proton: recent version: %v", err)
	}

	t.Log("TEST: beta version")
	versions[4].Version = 0
	if err := versions[4].WriteVersion(temp); err != nil {
		t.Errorf("test find proton: %v", err)
	}

	if err := versions[3].TestRecentVersion(temp); err != nil {
		t.Errorf("test find proton: beta version: %v", err)
	}

	t.Log("TEST: deleted version")
	if err := os.Remove(filepath.Join(temp, versions[3].Name, "version")); err != nil {
		t.Fatalf("test find proton: %v", err)
	}

	if err := versions[2].TestRecentVersion(temp); err != nil {
		t.Errorf("test find proton: deleted version: %v", err)
	}
}
