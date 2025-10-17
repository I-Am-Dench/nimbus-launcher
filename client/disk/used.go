package disk

import (
	"io/fs"
	"path/filepath"
)

func UsedSpace(directory string) (total uint64, err error) {
	filepath.Walk(directory, func(_ string, info fs.FileInfo, _ error) error {
		if !info.IsDir() {
			total += uint64(info.Size())
		}
		return nil
	})
	return total, err
}
