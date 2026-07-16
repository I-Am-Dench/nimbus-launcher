package patcher

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
)

// LastUpdate and Uptime are represented in RFC3339 format
type Status struct {
	XMLName        xml.Name  `xml:"ServerStatus"`
	DataCenterId   int       `xml:"DataCenterId"`
	Language       string    `xml:"Language"`
	LastUpdate     time.Time `xml:"LastUpdate"`
	Name           string    `xml:"Name"`
	Online         bool      `xml:"Online"`
	OnlineUsers    uint      `xml:"OnlineUsers"`
	OnlineUsersMax uint      `xml:"OnlineUsersMax"`
	Uptime         time.Time `xml:"Uptime"`
	Version        string    `xml:"Version"`
}

func (s Status) FormatUptimeSince(u time.Time) string {
	if s.Uptime.IsZero() {
		return ""
	}

	duration := u.Sub(s.Uptime)
	if duration < 0 {
		return ""
	}

	buf := []byte{}

	hours := int64(duration / time.Hour)

	days := hours / 24
	if days > 0 {
		buf = fmt.Append(buf, days, "d ")
	}

	if displayHours := hours % 24; displayHours > 0 || days > 0 {
		buf = fmt.Append(buf, displayHours, "h ")
	}

	minutes := int64(duration/time.Minute) % 60
	if minutes > 0 || hours > 0 {
		buf = fmt.Append(buf, minutes, "m ")
	}

	seconds := int64(duration/time.Second) % 60
	buf = fmt.Append(buf, seconds, "s")

	return string(buf)
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
		StatusList []Status `xml:"ServerStatus"`
	}{}

	reader, err := resources.Get(ctx, m.Status)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	xml.NewDecoder(reader).Decode(&statuses)
	return statuses.StatusList, nil
}
