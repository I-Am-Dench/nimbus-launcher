package patcher

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/I-Am-Dench/goverbuild/models/boot"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/resources"
)

type (
	Resources = resources.Resources
	Scheme    = resources.Scheme
)

type Config struct {
	ServiceUrl string          `json:"serviceUrl"`
	Config     json.RawMessage `json:"config"`
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

type Options struct {
	InstallDirectory string
	ServerId         string

	Log       Logger
	Resources Resources
}

type PatchEntry struct {
	Source, Destination string
}

type Patch interface {
	Run(context.Context) error
	Summary() []PatchEntry
}

type Patcher interface {
	GetBoot(packed bool) *boot.Config
	GetPatch(ctx context.Context, packed bool) (Patch, error)
}

type Environment interface {
	FormatMasterIndexUrl(serviceUrl string, scheme resources.Scheme) string

	NewPatcher(ctx context.Context, masterIndex MasterIndex, opt Options) (Patcher, error)
}

func GetMasterIndex(res Resources, uri string) (MasterIndex, error) {
	resource, err := res.Get(uri)
	if err != nil {
		return MasterIndex{}, fmt.Errorf("patcher: master index: %w", err)
	}
	defer resource.Close()

	masterIndex := MasterIndex{}
	if err := xml.NewDecoder(resource).Decode(&masterIndex); err != nil {
		return MasterIndex{}, fmt.Errorf("patcher: master index: %w", err)
	}

	return masterIndex, nil
}
