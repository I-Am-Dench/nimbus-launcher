package patcher

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"
)

type PatchOptions struct {
	InstallDirectory string
	ServerId         string

	Config json.RawMessage
}

type Authenticator interface {
	Authenticate(*http.Client) error
}

// type ResourceFunc = func(uri string) (io.ReadCloser, error)

type CredentialsFunc = func() (username string, password []byte, err error)

type Config struct {
	ForceRemoteResources bool
	// ResourceFunc         ResourceFunc
	CredentialsFunc CredentialsFunc
	CookieJar       http.CookieJar
}

var defaultCookieJar, _ = cookiejar.New(&cookiejar.Options{
	PublicSuffixList: publicsuffix.List,
})

var DefaultConfig = Config{
	CookieJar: defaultCookieJar,
}
