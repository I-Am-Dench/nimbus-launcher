package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	http_jar "net/http/cookiejar"
	"os"

	"github.com/I-Am-Dench/nimbus-launcher/app"
	"github.com/I-Am-Dench/nimbus-launcher/logger"
	"github.com/I-Am-Dench/nimbus-launcher/version"
	"golang.org/x/net/publicsuffix"

	cookiejar "github.com/juju/persistent-cookiejar"
)

const (
	SettingsDir = "./settings"
	Log         = "./current.log"
	Cookies     = "./cookies.json"
)

// Programs compiled with -H=windowsgui don't have
// an os.Stdout defined, so writes to the log writer
// fail before writing to a file.
//
// LaxMultiWriter is just a writer which ignores errors.
type LaxMultiWriter []io.Writer

func (l LaxMultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range l {
		w.Write(p)
	}
	return len(p), nil
}

type Saver interface {
	Save() error
}

func main() {
	file, err := os.Create(Log)
	if err != nil {
		log.Println(err)
	}

	var w io.Writer = os.Stdout
	if file != nil {
		defer file.Close()
		w = LaxMultiWriter{os.Stdout, file}
	}

	level := slog.LevelInfo
	if !version.Get().IsRelease || os.Getenv("NIMBUS_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(logger.NewHandler(w, &slog.HandlerOptions{
		Level: level,
	})))
	slog.Info("Starting Nimbus Launcher", "version", version.Get())

	if err := os.MkdirAll(SettingsDir, 0755); err != nil {
		slog.Error("Failed to create settings directory", "error", err)
	}

	var jar http.CookieJar
	jar, err = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
		Filename:         Cookies,
	})
	if err != nil {
		slog.Error("Failed to open cookie jar", "error", err)
		jar, _ = http_jar.New(&http_jar.Options{PublicSuffixList: publicsuffix.List})
	}
	defer func() {
		if saver, ok := jar.(Saver); ok {
			saver.Save()
		}
	}()

	a, err := app.New(SettingsDir, jar)
	if err != nil {
		slog.Error("Failed to create app", "error", err)
	}

	a.Start()
}
