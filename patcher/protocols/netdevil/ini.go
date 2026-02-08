package netdevil

import (
	"bufio"
	"io"
	"strings"
)

type Ini map[string]string

func ReadIni(r io.Reader) Ini {
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

	return m
}
