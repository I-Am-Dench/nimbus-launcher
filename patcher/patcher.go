package patcher

import (
	"context"
	"encoding/xml"

	"github.com/I-Am-Dench/goverbuild/archive"
	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
)

type PatchEntry struct {
	Source, Destination string
}

type Patch interface {
	Archive() *archive.Archive
	Summary() []PatchEntry
	Run(context.Context, undoer.Undoer) error
	Close() error
}

type Patcher interface {
	GetBoot(packed bool) *boot.Config
	GetPatch(ctx context.Context, packed bool) (Patch, error)
}

type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}

type Options struct {
	Resources origin.Resources
	Log       Logger

	ConfigUrl         string
	AuthenticationUrl string
	InstallDirectory  string
	ServerId          string
}

type MasterIndex struct {
	Authentication string `xml:"Authentication"`
	Config         struct {
		XMLName xml.Name `xml:"Config"`
		Type    string   `xml:"type,attr"`
		URL     string   `xml:",chardata"`
	}
	Status string `xml:"Status"`
}

type Environment interface {
	Locale() string
	GetMasterIndex(ctx context.Context, serviceUrl string, resources origin.Resources) (MasterIndex, error)
	NewPatcher(ctx context.Context, options Options) (Patcher, error)
}
