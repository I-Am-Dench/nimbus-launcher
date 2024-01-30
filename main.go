package main

import (
	"errors"
	"log"
	"os"

	"github.com/I-Am-Dench/lu-launcher/app"
	"github.com/I-Am-Dench/lu-launcher/resource"
	"github.com/I-Am-Dench/lu-launcher/version"
)

func main() {
	log.Printf("Starting Nimbus Launcher (%v)", version.Get())

	err := resource.InitializeSettings()
	if err != nil {
		log.Panicf("Settings initialization error: %v", err)
	}

	settings, err := resource.LauncherSettings()
	if err != nil {
		log.Println(err)
	}
	settings.Adjust()

	servers, err := resource.Servers()
	if err != nil {
		log.Println(err)
	}
	log.Printf("Loaded %d server configuration(s)\n", servers.Size())

	rejectedPatches, err := resource.PatchRejections()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Println(err)
	}
	log.Printf("Loaded %d patch rejection(s)\n", rejectedPatches.Amount())

	app := app.New(&settings, servers, rejectedPatches)
	app.Start()
}
