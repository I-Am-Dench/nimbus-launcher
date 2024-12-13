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
	ctx context.Context
}

func (*_file) Scheme() Scheme {
	return FileScheme
}

func (f *_file) Get(path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.FromSlash(filepath.Clean(path)))
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
	return &_file{ctx}
}

type _http struct {
	ctx    context.Context
	client *http.Client
}

func (*_http) Scheme() Scheme {
	return HttpScheme
}

func (h *_http) Get(uri string) (io.ReadCloser, error) {
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
	return &_http{ctx, client}
}

type _withRoot struct {
	Resources
	root string
}

func (r *_withRoot) Get(path string) (io.ReadCloser, error) {
	return r.Resources.Get(filepath.Join(r.root, path))
}

func WithRoot(r Resources, root string) Resources {
	return &_withRoot{r, root}
}

type _withUrl struct {
	Resources
	base string
}

func (r *_withUrl) Get(path string) (io.ReadCloser, error) {
	uri, err := url.JoinPath(r.base, filepath.ToSlash(path))
	if err != nil {
		return nil, fmt.Errorf("resources: %w", err)
	}
	return r.Resources.Get(uri)
}

func WithUrl(r Resources, base string) Resources {
	return &_withUrl{r, base}
}
