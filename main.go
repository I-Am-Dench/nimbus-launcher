package main

import (
	"log"

	"github.com/I-Am-Dench/lu-launcher/app"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

func main() {
	err := resource.InitializeSettings()
	if err != nil {
		log.Panicf("Settings initialization error: %v", err)
	}

	settings, err := resource.LauncherSettings()
	if err != nil {
		log.Println(err)
	}
	settings.Adjust()

	app := app.New(settings)
	app.Start()
}
