package patch

import "github.com/I-Am-Dench/lu-launcher/client"

type Patch interface {
	Version() string

	DownloadResources(Server, *RejectionList) error
	UpdateResources(Server, *RejectionList) error

	TransferResources(clientDirectory string, cache client.Cache, server Server) error
	TransferResourcesWithDependencies(clientDirectory string, cache client.Cache, server Server) error

	Summary() string
}
