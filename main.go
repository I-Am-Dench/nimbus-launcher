package main

import (
	"fmt"
	"log"
	"os"

	"github.com/I-Am-Dench/nimbus-launcher/app"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

const (
	SettingsDir = "settings"
)

func main() {
	fmt.Printf("Starting Nimbus Launcher (%v)\n", version.Get())

	if err := os.MkdirAll(SettingsDir, 0755); err != nil {
		log.Fatal(err)
	}

	a, err := app.New(SettingsDir)
	if err != nil {
		log.Fatal(err)
	}

	a.Start()
}
