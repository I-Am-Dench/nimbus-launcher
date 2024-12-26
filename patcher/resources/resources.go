package resources

import (
	"context"
	"errors"
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

var (
	ErrNotAuthenticated = errors.New("not authenticated")
)

type Resources interface {
	Scheme() Scheme
	Get(path string) (io.ReadCloser, error)
	Context() context.Context
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

func (f *_file) Context() context.Context {
	return f.ctx
}

func File(ctx context.Context) Resources {
	return &_file{ctx}
}

type CredentialsFunc func() (username string, password []byte, err error)

type HttpResources interface {
	Resources
	Client() *http.Client
}

type _http struct {
	ctx    context.Context
	client *http.Client
}

func (h *_http) Client() *http.Client {
	return h.client
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

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("resources: http: %w", ErrNotAuthenticated)
	}

	return nil, fmt.Errorf("resources: http: unhandled status: %s", response.Status)
}

func (h *_http) Context() context.Context {
	return h.ctx
}

func Http(ctx context.Context, client *http.Client) Resources {
	return &_http{ctx, client}
}

type _httpWithAuth struct {
	HttpResources

	credentials CredentialsFunc
	authUrl     string
}

func (h *_httpWithAuth) Authenticate() error {
	for {
		username, password, err := h.credentials()
		if err != nil {
			return fmt.Errorf("resources: http: authenticate: %w", err)
		}

		request, err := http.NewRequestWithContext(h.Context(), http.MethodPost, h.authUrl, nil)
		if err != nil {
			return fmt.Errorf("resources: http: authenticate: %w", err)
		}

		request.SetBasicAuth(username, string(password))

		response, err := h.HttpResources.Client().Do(request)
		if err != nil {
			return fmt.Errorf("resources: http: authenticate: %w", err)
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return nil
		}

		if response.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("resources: http: authenticate: unhandled status: %s", response.Status)
		}
	}
}

func (h *_httpWithAuth) Get(uri string) (io.ReadCloser, error) {
	for {
		reader, err := h.HttpResources.Get(uri)
		if err == nil {
			return reader, nil
		}

		if !errors.Is(err, ErrNotAuthenticated) {
			return nil, err
		}

		if err := h.Authenticate(); err != nil {
			return nil, err
		}
	}
}

func WithAuthentication(r HttpResources, credentials CredentialsFunc, authUrl string) Resources {
	if credentials == nil {
		panic(fmt.Errorf("credentials cannot be nil"))
	}

	return &_httpWithAuth{r, credentials, authUrl}
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
