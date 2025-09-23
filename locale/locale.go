package locale

import "strings"

var locales = map[string]string{
	"en_us": "US English",
	"en_gb": "UK English",
	"de_de": "German (Deutsch)",
}

func GetName(id string) string {
	name, ok := locales[strings.ToLower(id)]
	if !ok {
		return id
	}
	return name
}

func List() []string {
	return []string{
		"en_US",
		"en_GB",
		"de_DE",
	}
}
