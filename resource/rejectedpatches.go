package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type RejectedPatches struct {
	rejections map[string][]string
}

func NewRejectedPatches() RejectedPatches {
	return RejectedPatches{
		rejections: map[string][]string{},
	}
}

func (rejected *RejectedPatches) Load() error {
	data, err := os.ReadFile(filepath.Join(settingsDir, "rejectedPatches.json"))
	if err != nil {
		return fmt.Errorf("cannot read rejectedPatches.json: %v", err)
	}

	err = json.Unmarshal(data, &rejected.rejections)
	if err != nil {
		return fmt.Errorf("cannot unmarshal rejectedPatches.json: %v", err)
	}

	return nil
}

func (rejected *RejectedPatches) Save() error {
	data, err := json.MarshalIndent(rejected.rejections, "", "    ")
	if err != nil {
		return fmt.Errorf("cannot marshal rejectedPatches.json: %v", err)
	}

	err = os.WriteFile(filepath.Join(settingsDir, "rejectedPatches.json"), data, 0755)
	if err != nil {
		return fmt.Errorf("cannot write rejectedPatches.json: %v", err)
	}

	return nil
}

func (rejected *RejectedPatches) Amount() int {
	sum := 0

	for _, versions := range rejected.rejections {
		if versions == nil {
			continue
		}

		sum += len(versions)
	}

	return sum
}

func (rejected *RejectedPatches) Add(server *Server, version string) error {
	if server == nil || len(server.Id) <= 0 || len(version) <= 0 {
		return nil
	}

	versions, ok := rejected.rejections[server.Id]
	if !ok {
		versions = []string{}
	}

	versions = append(versions, version)
	rejected.rejections[server.Id] = versions

	return rejected.Save()
}

func (rejected *RejectedPatches) IsRejected(server *Server, version string) bool {
	if server == nil || len(server.Id) <= 0 || len(version) <= 0 {
		return false
	}

	versions, ok := rejected.rejections[server.Id]
	if !ok {
		return false
	}

	for _, v := range versions {
		if v == version {
			return true
		}
	}

	return false
}
