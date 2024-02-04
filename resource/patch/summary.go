package patch

type Summary struct {
	CurrentVersion    string   `json:"currentVersion"`
	AvailableVersions []string `json:"availableVersions"`
}
