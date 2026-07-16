package netdevil

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Ini map[string]string

func (i Ini) PatcherExeVersion() (version string, extended bool) {
	version, ok := i["patcherexeversion"]
	if !ok {
		return "", false
	}
	return version, len(version) > 0 && version[0] == '+'
}

func (i Ini) List(name string) []string {
	l := []string{}

	value, ok := i[name]
	if !ok {
		return l
	}

	for elem := range strings.SplitSeq(value, ",") {
		if s := strings.TrimSpace(elem); len(s) > 0 {
			l = append(l, s)
		}
	}
	return l
}

func ReadIni(r io.Reader) (Ini, error) {
	scanner := bufio.NewScanner(r)

	m := Ini{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && line[0] == '#' {
			continue
		}

		key, value, _ := strings.Cut(line, "=")
		if len(key) == 0 {
			continue
		}

		m[key] = strings.Trim(value, "\" ")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ini: %v", err)
	}
	return m, nil
}
