package main

import (
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/I-Am-Dench/nimbus-launcher/app"
	"github.com/I-Am-Dench/nimbus-launcher/logger"
	"github.com/I-Am-Dench/nimbus-launcher/version"
)

const (
	SettingsDir = "./settings"
	Log         = "./current.log"
)

func main() {
	file, err := os.Create(Log)
	if err != nil {
		log.Println(err)
	}

	var w io.Writer = os.Stdout
	if file != nil {
		defer file.Close()
		w = io.MultiWriter(os.Stdout, file)
	}

	slog.SetDefault(slog.New(logger.NewHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	slog.Info("Starting Nimbus Launcher", "version", version.Get())

	if err := os.MkdirAll(SettingsDir, 0755); err != nil {
		slog.Error("Failed to create settings directory", "error", err)
	}

	a, err := app.New(SettingsDir)
	if err != nil {
		slog.Error("Failed to create app", "error", err)
	}

	a.Start()
}
