package patch

import (
	"net/http"

	"github.com/I-Am-Dench/lu-launcher/ldf"
)

type Server interface {
	// Returns the server's ID.
	Id() string

	// Returns the directory where patch content should be saved to.
	DownloadDir() string

	// Returns the patch from the server corresponding to the version.
	GetPatch(version string) (Patch, error)

	// Makes an HTTP request to the server where the parameter, elem, contains the components
	// which are appended to the requested path.
	RemoteGet(elem ...string) (*http.Response, error)

	// Updates contents of the server's boot.cfg.
	SetBootConfig(*ldf.BootConfig) error

	// Updates the protocol of the server uses for patching.
	SetPatchProtocol(protocol string)
}
