package patch

import (
	"net/http"

	"github.com/I-Am-Dench/lu-launcher/ldf"
)

type Server interface {
	Id() string
	DownloadDir() string

	GetPatch(version string) (Patch, error)
	RemoteGet(elem ...string) (*http.Response, error)

	SetBootConfig(*ldf.BootConfig) error
	SetPatchProtocol(protocol string)
}
