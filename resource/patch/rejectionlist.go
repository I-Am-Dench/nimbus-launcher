package patch

import (
	"encoding/json"
	"fmt"
	"os"
)

type RejectionList struct {
	path string
	m    map[string][]string
}

func NewRejectionList(path string) *RejectionList {
	return &RejectionList{
		path: path,
		m:    make(map[string][]string),
	}
}

func (rejections *RejectionList) Load() error {
	data, err := os.ReadFile(rejections.path)
	if err != nil {
		return fmt.Errorf("load rejections: cannot load file: %w", err)
	}

	err = json.Unmarshal(data, &rejections.m)
	if err != nil {
		return fmt.Errorf("load rejections: cannot unmarshal file: %w", err)
	}

	return nil
}

func (rejections *RejectionList) Save() error {
	data, err := json.MarshalIndent(rejections.m, "", "    ")
	if err != nil {
		return fmt.Errorf("save rejections: cannot marshal contents: %w", err)
	}

	err = os.WriteFile(rejections.path, data, 0755)
	if err != nil {
		return fmt.Errorf("save rejections: cannot save data: %w", err)
	}

	return nil
}

func (rejections *RejectionList) Amount() int {
	sum := 0

	for _, versions := range rejections.m {
		if versions == nil {
			continue
		}

		sum += len(versions)
	}

	return sum
}

func (rejections *RejectionList) Add(server Server, version string) error {
	if server == nil || len(server.Id()) == 0 || len(version) == 0 {
		return nil
	}

	versions, ok := rejections.m[server.Id()]
	if !ok {
		versions = []string{}
	}

	versions = append(versions, version)
	rejections.m[server.Id()] = versions

	return rejections.Save()
}

func (rejections *RejectionList) IsRejected(server Server, version string) bool {
	if server == nil || len(server.Id()) == 0 || len(version) == 0 {
		return false
	}

	versions, ok := rejections.m[server.Id()]
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
