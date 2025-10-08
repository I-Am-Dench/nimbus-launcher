// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cookiejar

import (
	"strings"
	"unicode"
)

// asciiIsPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func asciiIsPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

// Is returns whether s is ASCII.
func asciiIs(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// ToLower returns the lowercase version of s if s is ASCII and printable.
func asciiToLower(s string) (lower string, ok bool) {
	if !asciiIsPrint(s) {
		return "", false
	}
	return strings.ToLower(s), true
}
