package server

import (
	"encoding/xml"
	"fmt"
	"os"
)

type XML struct {
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

func LoadXML(name string) (XML, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return XML{}, fmt.Errorf("cannot read server XML \"%s\": %v", name, err)
	}

	server := XML{}
	err = xml.Unmarshal(data, &server)
	if err != nil {
		return XML{}, fmt.Errorf("cannot unmarshal server XML \"%s\": %v", name, err)
	}

	return server, nil
}
