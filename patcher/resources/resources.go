package resources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Scheme int

const (
	FileScheme = Scheme(iota)
	HttpScheme
)

type Resources interface {
	Scheme() Scheme
	SetRoot(uri string)

	Get(path string) (io.ReadCloser, error)
}

func ParseScheme(uri string) (Scheme, string, error) {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		return 0, "", fmt.Errorf("resources: invalid uri: %w", err)
	}

	switch url.Scheme {
	default:
		return 0, "", fmt.Errorf("resources: unknown scheme: %s", url.Scheme)
	case "file":
		return FileScheme, filepath.FromSlash(strings.TrimPrefix(url.Path, "/")), nil
	case "http", "https":
		return HttpScheme, uri, nil
	}
}

type _file struct {
	ctx  context.Context
	root string
}

func (*_file) Scheme() Scheme {
	return FileScheme
}

func (f *_file) SetRoot(uri string) {
	f.root = uri
}

func (f *_file) Get(path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.FromSlash(filepath.Join(f.root, filepath.Clean(path))))
	if err != nil {
		return nil, fmt.Errorf("resources: file: %w", err)
	}

	select {
	case <-f.ctx.Done():
		file.Close()
		return nil, f.ctx.Err()
	default:
		return file, nil
	}
}

func File(ctx context.Context) Resources {
	return &_file{ctx, ""}
}

type _http struct {
	ctx    context.Context
	client *http.Client

	base string
}

func (*_http) Scheme() Scheme {
	return HttpScheme
}

func (h *_http) SetRoot(uri string) {
	h.base = uri
}

func (h *_http) Get(uri string) (io.ReadCloser, error) {
	if len(h.base) > 0 {
		var err error
		uri, err = url.JoinPath(h.base, filepath.ToSlash(uri))
		if err != nil {
			return nil, fmt.Errorf("resources: http: %w", err)
		}
	}

	request, err := http.NewRequestWithContext(h.ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("resources: http: %w", err)
	}

	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("resources: http: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response.Body, nil
	}

	// Allows the client to reuse the connection when using Keep-Alive
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	return nil, fmt.Errorf("resources: http: unhandled status: %s", response.Status)
}

func Http(ctx context.Context, client *http.Client) Resources {
	return &_http{ctx, client, ""}
}
