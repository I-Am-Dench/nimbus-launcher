//go:build unix
// +build unix

package disk

import (
	"golang.org/x/sys/unix"
)

func FreeSpace(directory string) (freeBytes uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(directory, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
