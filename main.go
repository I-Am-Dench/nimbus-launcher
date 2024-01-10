package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/I-Am-Dench/lu-launcher/app"
	"github.com/I-Am-Dench/lu-launcher/resource"
)

const (
	crashlogNameFormat      = "2006-01-02-150405.log"
	crashlogTimestampFormat = "2006-01-02 at 15:04:05"
)

func WriteCrashLog(crashLog string) {
	err := os.MkdirAll("crashlogs", 0755)
	if err != nil {
		log.Panicln(err)
	}

	now := time.Now()

	filename := fmt.Sprintf("crashlogs/%v", now.Format(crashlogNameFormat))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()

	file.WriteString("NIMBUS LAUNCHER CRASH\n")
	file.WriteString("Time: ")
	file.WriteString(now.Format(crashlogTimestampFormat))
	file.WriteString("\n\nOS: ")
	file.WriteString(runtime.GOOS)
	file.WriteString("\nARCH: ")
	file.WriteString(runtime.GOARCH)
	file.WriteString("\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n\n")
	file.WriteString(crashLog)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recover: %v", r)
			WriteCrashLog(fmt.Sprintf("%v", r))
		}
	}()

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
	if err != nil {
		log.Println(err)
	}
	log.Printf("Loaded %d patch rejection(s)\n", rejectedPatches.Amount())

	app := app.New(&settings, servers, rejectedPatches)
	if err := app.Start(); err != nil {
		panic(err)
	}
}
