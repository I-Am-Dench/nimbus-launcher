package patch_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/I-Am-Dench/lu-launcher/ldf"
	"github.com/I-Am-Dench/lu-launcher/resource/patch"
	"github.com/I-Am-Dench/lu-launcher/resource/server"
)

type environment struct {
	Dir string

	PatchServer  *http.Server
	ServerConfig *server.Server
	Rejections   *patch.RejectionList
}

func (env *environment) ClientDir() string {
	return filepath.Join(env.Dir, "client")
}

type fileSystem map[string][]byte

func (fs fileSystem) Init(dir string, t *testing.T) {
	t.Helper()

	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("init fs: %v", err)
	}

	for relativePath, data := range fs {
		path := filepath.Join(dir, relativePath)
		t.Logf("creating client file: %s", path)

		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("init fs: %v", err)
		}

		err = os.WriteFile(path, data, 0755)
		if err != nil {
			t.Fatalf("init fs: %v", err)
		}
	}
}

func newPatchServer(t *testing.T, ctx context.Context, fs fileSystem) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("[PATCH SERVER] {%s} %s", r.Method, r.URL.Path)

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		data, ok := fs[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	return &http.Server{
		Addr:    "127.0.0.1:3000",
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
}

func newServerConfiguration(dir string) *server.Server {
	return server.New(server.Config{
		SettingsDir:   dir,
		DownloadDir:   filepath.Join(dir, "patches"),
		Name:          "",
		PatchProtocol: "http",
		Config: &ldf.BootConfig{
			PatchServerIP:   "127.0.0.1",
			PatchServerPort: 3000,
			PatchServerDir:  "patches",
		},
	})
}

func setup(t *testing.T, serverFS fileSystem) (*environment, func()) {
	t.Helper()

	dir, err := os.MkdirTemp(".", "patch_test*.tmp")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Logf("Using temp dir \"%s\"", dir)

	env := &environment{
		Dir: dir,
	}

	ctx := context.Background()
	env.PatchServer = newPatchServer(t, ctx, serverFS)
	env.PatchServer.RegisterOnShutdown(func() {
		t.Logf("Patch server shutdown.")
	})

	env.ServerConfig = newServerConfiguration(dir)
	err = os.MkdirAll(filepath.Join(dir, "servers"), 0755) // directory where servers' boot.cfgs are stored
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	env.Rejections = patch.NewRejectionList(filepath.Join(dir, "rejections.json"))

	return env, func() {
		env.PatchServer.Shutdown(ctx)

		err := os.RemoveAll(dir)
		if err != nil {
			t.Log(err)
		}
	}
}
