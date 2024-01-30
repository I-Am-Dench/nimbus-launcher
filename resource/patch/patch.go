package patch

import "github.com/I-Am-Dench/lu-launcher/client"

type Patch interface {
	// Returns the version of this patch
	Version() string

	// Downloads resources needed by the patch to the path returned by Server.DownloadDir().
	DownloadResources(Server, *RejectionList) error

	// Updates the Server's configuration.
	UpdateResources(Server, *RejectionList) error

	// Transfers resources downloaded by the patch into the clientDirectory.
	TransferResources(clientDirectory string, resources client.Resources, server Server) error

	// Transfers resources downloaded by the patch into the clientDirectory, transferring dependencies' resources
	// if possible.
	TransferResourcesWithDependencies(clientDirectory string, resources client.Resources, server Server) error

	// A stringified summary of this patch.
	Summary() string
}
