// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cookiejar

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

type Options = cookiejar.Options

type Jar struct {
	name   string
	psList cookiejar.PublicSuffixList

	mu      sync.RWMutex
	entries map[string]map[string]entry
}

func New(name string, o *Options) (*Jar, error) {
	if o == nil {
		o = &Options{}
	}

	jar := &Jar{
		name:   name,
		psList: o.PublicSuffixList,

		entries: make(map[string]map[string]entry),
	}
	if err := jar.Load(); err != nil {
		return nil, err
	}

	return jar, nil
}

func (j *Jar) Load() error {
	file, err := os.Open(j.name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("cookiejar: load: %v", err)
	}
	defer file.Close()

	persist := []entry{}
	if err := json.NewDecoder(file).Decode(&persist); err != nil {
		return fmt.Errorf("cookiejar: load: %v", err)
	}

	entries := make(map[string]map[string]entry)
	for _, e := range persist {
		etld, err := canonicalHost(e.Domain)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		submap, ok := entries[etld]
		if !ok {
			submap = make(map[string]entry)
			entries[etld] = submap
		}

		submap[e.id()] = e
	}

	j.entries = entries

	return nil
}

func (j *Jar) Store() error {
	persist := []entry{}
	for _, submap := range j.entries {
		for _, e := range submap {
			if !e.Expires.IsZero() {
				persist = append(persist, e)
			}
		}
	}

	file, err := os.Create(j.name)
	if err != nil {
		return fmt.Errorf("cookiejar: store: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(persist); err != nil {
		return fmt.Errorf("cookiejar: store: %v", err)
	}

	return nil
}

type entry struct {
	Name     string        `json:"name"`
	Value    string        `json:"value"`
	Quoted   bool          `json:"quoted"`
	Domain   string        `json:"domain"`
	Path     string        `json:"path"`
	SameSite http.SameSite `json:"sameSite"`
	Secure   bool          `json:"secure"`
	HttpOnly bool          `json:"httpOnly"`
	HostOnly bool          `json:"hostOnly"`
	Expires  time.Time     `json:"expires"`
	Created  time.Time     `json:"created"`
}

// id returns the domain;path;name triple of e as an id.
func (e *entry) id() string {
	return strings.Join([]string{e.Domain, e.Path, e.Name}, ";")
}

// shouldSend determines whether e's cookie qualifies to be included in a
// request to host/path. It is the caller's responsibility to check if the
// cookie is expired.
func (e *entry) shouldSend(https bool, host, path string) bool {
	return e.domainMatch(host) && e.pathMatch(path) && (https || !e.Secure)
}

// domainMatch checks whether e's Domain allows sending e back to host.
// It differs from "domain-match" of RFC 6265 section 5.1.3 because we treat
// a cookie with an IP address in the Domain always as a host cookie.
func (e *entry) domainMatch(host string) bool {
	if e.Domain == host {
		return true
	}
	return !e.HostOnly && hasDotSuffix(host, e.Domain)
}

// pathMatch implements "path-match" according to RFC 6265 section 5.1.4.
func (e *entry) pathMatch(requestPath string) bool {
	if requestPath == e.Path {
		return true
	}
	if strings.HasPrefix(requestPath, e.Path) {
		if e.Path[len(e.Path)-1] == '/' {
			return true // The "/any/" matches "/any/path" case.
		} else if requestPath[len(e.Path)] == '/' {
			return true // The "/any" matches "/any/path" case.
		}
	}
	return false
}

// hasDotSuffix reports whether s ends in "."+suffix.
func hasDotSuffix(s, suffix string) bool {
	return len(s) > len(suffix) && s[len(s)-len(suffix)-1] == '.' && s[len(s)-len(suffix):] == suffix
}

// Cookies implements the Cookies method of the [http.CookieJar] interface.
//
// It returns an empty slice if the URL's scheme is not HTTP or HTTPS.
func (j *Jar) Cookies(u *url.URL) []*http.Cookie {
	return j.cookies(u, time.Now())
}

// cookies is like Cookies but takes the current time as a parameter.
func (j *Jar) cookies(u *url.URL, now time.Time) (cookies []*http.Cookie) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return cookies
	}

	j.mu.RLock()
	defer j.mu.RUnlock()

	host, err := canonicalHost(u.Hostname())
	if err != nil {
		return cookies
	}
	etld := eTldPlus1(host, j.psList)

	submap := j.entries[etld]
	if submap == nil {
		return cookies
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	selected := []entry{}

	modified := false
	for id, e := range submap {
		if !e.Expires.IsZero() && !e.Expires.After(now) {
			delete(submap, id)
			modified = true
			continue
		}

		if e.shouldSend(u.Scheme == "https", host, path) {
			selected = append(selected, e)
		}
	}
	if modified {
		if len(submap) == 0 {
			delete(j.entries, etld)
		} else {
			j.entries[etld] = submap
		}

		if err := j.Store(); err != nil {
			slog.Error(err.Error())
		}
	}

	slices.SortFunc(selected, func(a, b entry) int {
		if r := cmp.Compare(a.Path, b.Path); r != 0 {
			return r
		}
		return a.Created.Compare(b.Created)
	})
	for _, e := range selected {
		cookies = append(cookies, &http.Cookie{Name: e.Name, Value: e.Value, Quoted: e.Quoted})
	}

	return cookies
}

func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.setCookies(u, cookies, time.Now())
}

func (j *Jar) setCookies(u *url.URL, cookies []*http.Cookie, now time.Time) {
	if len(cookies) == 0 {
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return
	}

	host, err := canonicalHost(u.Hostname())
	if err != nil {
		return
	}
	defPath := defaultPath(u.Path)

	etld := eTldPlus1(host, j.psList)

	j.mu.Lock()
	defer j.mu.Unlock()

	submap := j.entries[etld]

	modified := false
	for _, cookie := range cookies {
		e, remove, err := j.newEntry(cookie, now, defPath, host)
		if err != nil {
			continue
		}

		id := e.id()
		if remove {
			if submap != nil {
				if _, ok := submap[id]; ok {
					delete(submap, id)
					modified = true
				}
			}
			continue
		}
		if submap == nil {
			submap = make(map[string]entry)
		}

		if old, ok := submap[id]; ok {
			e.Created = old.Created
		} else {
			e.Created = time.Now()
		}

		submap[id] = e
		modified = true
	}

	if modified {
		if len(submap) == 0 {
			delete(j.entries, etld)
		} else {
			j.entries[etld] = submap
		}

		if err := j.Store(); err != nil {
			slog.Error(err.Error())
		}
	}
}

func canonicalHost(host string) (string, error) {
	if isIP(host) {
		return host, nil
	}

	host = strings.TrimSuffix(host, ".")
	encoded, err := toASCII(host)
	if err != nil {
		return "", err
	}
	// We know this is ascii, no need to check
	lower, _ := asciiToLower(encoded)
	return lower, nil
}

func eTldPlus1(host string, psl cookiejar.PublicSuffixList) string {
	if isIP(host) {
		return host
	}

	var i int
	if psl == nil {
		i = strings.LastIndex(host, ".")
		if i <= 0 {
			return host
		}
	} else {
		suffix := psl.PublicSuffix(host)
		if suffix == host {
			return host
		}
		i = len(host) - len(suffix)
		if i <= 0 || host[i-1] != '.' {
			// The provided public suffix list psl is broken.
			// Storing cookies under host is a safe stopgap.
			return host
		}
		// Only len(suffix) is used to determine the jar key from
		// here on, so it is okay if psl.PublicSuffix("www.buggy.psl")
		// returns "com" as the jar key is generated from host.
	}
	prevDot := strings.LastIndex(host[:i-1], ".")
	return host[prevDot+1:]
}

// isIP reports whether host is an IP address
func isIP(host string) bool {
	if strings.ContainsAny(host, ":%") {
		// Probable IPv6 address.
		// Hostnames can't contain : or %, so this is definitely not a valid host.
		// Treating it as an IP is the more conservative option, and avoids the risk
		// of interpreting ::1%.www.example.com as a subdomain of www.example.com.
		return true
	}
	return net.ParseIP(host) != nil
}

// defaultPath returns the directory part of a URL's path according to
// RFC 6265 section 5.1.4.
func defaultPath(path string) string {
	if len(path) == 0 || path[0] != '/' {
		return "/" // Path is empty or malformed.
	}

	i := strings.LastIndex(path, "/") // Path starts with "/", so i != -1.
	if i == 0 {
		return "/" // Path has the form "/abc".
	}
	return path[:i] // Path is either of form "/abc/xyz" or "/abc/xyz/".
}

// newEntry creates an entry from an http.Cookie c. now is the current time and
// is compared to c.Expires to determine deletion of c. defPath and host are the
// default-path and the canonical host name of the URL c was received from.
//
// remove records whether the jar should delete this cookie, as it has already
// expired with respect to now. In this case, e may be incomplete, but it will
// be valid to call e.id (which depends on e's Name, Domain and Path).
//
// A malformed c.Domain will result in an error.
func (j *Jar) newEntry(c *http.Cookie, now time.Time, defPath string, host string) (e entry, remove bool, err error) {
	e.Name = c.Name

	if c.Path == "" || c.Path[0] != '/' {
		e.Path = defPath
	} else {
		e.Path = c.Path
	}

	e.Domain, e.HostOnly, err = j.domainAndType(host, c.Domain)
	if err != nil {
		return e, false, err
	}

	if c.MaxAge < 0 {
		return e, true, nil
	}

	// MaxAge takes prcedence over Expires.
	e.Expires = c.Expires
	if c.MaxAge > 0 {
		e.Expires = now.Add(time.Duration(c.MaxAge) * time.Second)
	}

	e.Value = c.Value
	e.Quoted = c.Quoted
	e.Secure = c.Secure
	e.HttpOnly = c.HttpOnly
	e.SameSite = c.SameSite

	return e, false, nil
}

var (
	errIllegalDomain   = errors.New("cookiejar: illegal cookie domain attribute")
	errMalformedDomain = errors.New("cookiejar: malformed cookie domain attribute")
)

// domainAndType determines the cookie's domain and hostOnly attribute.
func (j *Jar) domainAndType(host, domain string) (string, bool, error) {
	if domain == "" {
		// No domain attribute in the SetCookie header indicates a
		// host cookie.
		return host, true, nil
	}

	if isIP(host) {
		// RFC 6265 is not super clear here, a sensible interpretation
		// is that cookies with an IP address in the domain-attribute
		// are allowed.

		// RFC 6265 section 5.2.3 mandates to strip an optional leading
		// dot in the domain-attribute before processing the cookie.
		//
		// Most browsers don't do that for IP addresses, only curl
		// (version 7.54) and IE (version 11) do not reject a
		//     Set-Cookie: a=1; domain=.127.0.0.1
		// This leading dot is optional and serves only as hint for
		// humans to indicate that a cookie with "domain=.bbc.co.uk"
		// would be sent to every subdomain of bbc.co.uk.
		// It just doesn't make sense on IP addresses.
		// The other processing and validation steps in RFC 6265 just
		// collapse to:
		if host != domain {
			return "", false, errIllegalDomain
		}

		// According to RFC 6265 such cookies should be treated as
		// domain cookies.
		// As there are no subdomains of an IP address the treatment
		// according to RFC 6265 would be exactly the same as that of
		// a host-only cookie. Contemporary browsers (and curl) do
		// allows such cookies but treat them as host-only cookies.
		// So do we as it just doesn't make sense to label them as
		// domain cookies when there is no domain; the whole notion of
		// domain cookies requires a domain name to be well defined.
		return host, true, nil
	}

	// From here on: If the cookie is valid, it is a domain cookie (with
	// the one exception of a public suffix below).
	// See RFC 6265 section 5.2.3.
	domain = strings.TrimPrefix(domain, ".")

	if len(domain) == 0 || domain[0] == '.' {
		// Received either "Domain=." or "Domain=..some.thing",
		// both are illegal.
		return "", false, errMalformedDomain
	}

	domain, isASCII := asciiToLower(domain)
	if !isASCII {
		// Received non-ASCII domain, e.g. "perché.com" instead of "xn--perch-fsa.com"
		return "", false, errMalformedDomain
	}

	if domain[len(domain)-1] == '.' {
		// We received stuff like "Domain=www.example.com.".
		// Browsers do handle such stuff (actually differently) but
		// RFC 6265 seems to be clear here (e.g. section 4.1.2.3) in
		// requiring a reject.  4.1.2.3 is not normative, but
		// "Domain Matching" (5.1.3) and "Canonicalized Host Names"
		// (5.1.2) are.
		return "", false, errMalformedDomain
	}

	// See RFC 6265 section 5.3 #5.
	if j.psList != nil {
		if ps := j.psList.PublicSuffix(domain); ps != "" && !hasDotSuffix(domain, ps) {
			if host == domain {
				// This is the one exception in which a cookie
				// with a domain attribute is a host cookie.
				return host, true, nil
			}
			return "", false, errIllegalDomain
		}
	}

	// The domain must domain-match host: www.mycompany.com cannot
	// set cookies for .ourcompetitors.com.
	if host != domain && !hasDotSuffix(host, domain) {
		return "", false, errIllegalDomain
	}

	return domain, false, nil
}
