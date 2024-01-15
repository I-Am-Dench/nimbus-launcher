package version

import "fmt"

type Version struct {
	Major, Minor, Patch uint

	isRelease bool
}

func Get() Version {
	return v
}

func (v Version) String() string {
	if v.isRelease {
		return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	} else {
		return "standalone"
	}
}
