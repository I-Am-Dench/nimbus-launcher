//go:build windows
// +build windows

package disk

import "golang.org/x/sys/windows"

func FreeSpace(directory string) (freeBytes uint64, err error) {
	s, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return 0, err
	}
	return freeBytes, windows.GetDiskFreeSpaceEx(s, &freeBytes, nil, nil)
}
