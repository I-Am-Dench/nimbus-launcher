package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	http_jar "net/http/cookiejar"
	"os"

	"github.com/I-Am-Dench/nimbus-launcher/app"
	"github.com/I-Am-Dench/nimbus-launcher/app/cookiejar"
	"github.com/I-Am-Dench/nimbus-launcher/logger"
	"github.com/I-Am-Dench/nimbus-launcher/version"
	"golang.org/x/net/publicsuffix"
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

	var jar http.CookieJar
	jar, err = cookiejar.New("cookies.json", &cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		slog.Error("Failed to open cookie jar", "error", err)
		jar, _ = http_jar.New(&http_jar.Options{PublicSuffixList: publicsuffix.List})
	}
	defer func() {
		if closer, ok := jar.(io.Closer); ok {
			closer.Close()
		}
	}()

	a, err := app.New(SettingsDir, jar)
	if err != nil {
		slog.Error("Failed to create app", "error", err)
	}

	a.Start()
}
