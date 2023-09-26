package resource

import (
	"encoding/xml"
	"fmt"
	"os"
)

type ServerXML struct {
	XMLName xml.Name `xml:"server"`
	Name    string   `xml:"name"`
	Patch   struct {
		XMLName  xml.Name `xml:"patch"`
		Token    string   `xml:"token"`
		Protocol string   `xml:"protocol"`
	} `xml:"patch"`
	Boot struct {
		Text string `xml:",innerxml"`
	} `xml:"boot"`
}

func LoadXML(name string) (ServerXML, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return ServerXML{}, fmt.Errorf("cannot read server XML \"%s\": %v", name, err)
	}

	server := ServerXML{}
	err = xml.Unmarshal(data, &server)
	if err != nil {
		return ServerXML{}, fmt.Errorf("cannot unmarshal server XML \"%s\": %v", name, err)
	}

	return server, nil
}
