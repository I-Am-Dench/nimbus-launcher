package version

import (
	"fmt"
	"runtime/debug"
)

type Version struct {
	Major, Minor, Patch uint

	IsRelease bool
}

var revision string

func init() {
	info, _ := debug.ReadBuildInfo()
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
			return
		}
	}
}

func (v Version) String() string {
	if v.IsRelease {
		return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	}

	if len(revision) == 0 {
		return "standalone"
	}

	return revision
}

func (v Version) Name() string {
	if v.IsRelease {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	} else {
		return "Standalone"
	}
}

func Get() Version {
	return v
}

func Revision() string {
	return revision
}
