package patch

import (
	"fmt"
	"regexp"
)

var versionPattern = regexp.MustCompile(`^(v|V)?[0-9]+\.[0-9]+\.[0-9]+([0-9a-zA-Z_.-]+)?$`)

func ValidateVersionName(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version name \"%s\": version must match `%v`", version, versionPattern)
	}
	return nil
}
