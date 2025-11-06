package patcher

import (
	"context"
	"encoding/xml"
	"time"

	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

type Status struct {
	XMLName        xml.Name  `xml:"ServerStatus"`
	DataCenterId   int       `xml:"DataCenterId"`
	Language       string    `xml:"Language"`
	LastUpdate     time.Time `xml:"LastUpdate"`
	Name           string    `xml:"Name"`
	Online         bool      `xml:"Online"`
	OnlineUsers    uint      `xml:"OnlineUsers"`
	OnlineUsersMax uint      `xml:"OnlineUsersMax"`
	Uptime         uint64    `xml:"Uptime"`
	Version        string    `xml:"Version"`
}

type MasterIndex struct {
	Authentication string `xml:"Authentication"`
	UniverseConfig struct {
		XMLName xml.Name `xml:"Config"`
		Type    string   `xml:"type,attr"`
		URL     string   `xml:",chardata"`
	}
	Status string `xml:"Status"`
}

func (m MasterIndex) GetStatusList(ctx context.Context, resources origin.Resources) ([]Status, error) {
	statuses := struct {
		XMLName    xml.Name `xml:"ArrayOfServerStatus"`
		StatusList []Status
	}{}

	reader, err := resources.Get(ctx, m.Status)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	xml.NewDecoder(reader).Decode(&statuses)
	return statuses.StatusList, nil
}
