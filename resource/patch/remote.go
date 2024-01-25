package patch

import "net/http"

type Remote interface {
	GetPatch(version string) Patch
	RemoteGet(elem ...string) (*http.Response, error)
}
