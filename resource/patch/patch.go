package patch

import "github.com/I-Am-Dench/lu-launcher/client"

type Patch interface {
	Version() string

	DownloadResources(Server, *RejectionList) error
	UpdateResources(Server, *RejectionList) error

	TransferResources(clientDirectory string, resources client.Resources, server Server) error
	TransferResourcesWithDependencies(clientDirectory string, resources client.Resources, server Server) error

	Summary() string
}
