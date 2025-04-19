package remote

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

	"github.com/I-Am-Dench/nimbus-launcher/version"
)

type Scheme int

const (
	FileScheme = Scheme(iota)
	HttpScheme
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
)

func ParseScheme(uri string) (Scheme, string, error) {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		return 0, "", fmt.Errorf("parse scheme: invalid uri: %w", err)
	}

	switch url.Scheme {
	default:
		return 0, "", fmt.Errorf("parse scheme: unknown scheme: %s", url.Scheme)
	case "file":
		return FileScheme, filepath.FromSlash(strings.TrimPrefix(url.Path, "/")), nil
	case "http", "https":
		return HttpScheme, uri, nil
	}
}

type Resources interface {
	Scheme() Scheme
	Get(ctx context.Context, path string) (io.ReadCloser, error)
}

type _file struct{}

func (*_file) Scheme() Scheme {
	return FileScheme
}

func (*_file) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.FromSlash(filepath.Clean(path)))
	if err != nil {
		return nil, fmt.Errorf("resources: file: %w", err)
	}

	select {
	case <-ctx.Done():
		file.Close()
		return nil, ctx.Err()
	default:
		return file, nil
	}
}

func File() Resources {
	return &_file{}
}

var UserAgent = "NimbusLauncher/" + version.Get().Name()

type CredentialsFunc func() (username string, password []byte, err error)

type HttpResources interface {
	Resources
	Client() *http.Client
}

type _http struct {
	client *http.Client
}

func (*_http) Scheme() Scheme {
	return HttpScheme
}

func (h *_http) Client() *http.Client {
	return h.client
}

func (h *_http) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("resources: http: %w", err)
	}

	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("User-Agent", UserAgent)

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

func Http(client *http.Client) Resources {
	return &_http{client}
}

type _httpWithAuth struct {
	HttpResources

	credentials CredentialsFunc
	authUrl     string
}

func (h *_httpWithAuth) Authenticate(ctx context.Context) error {
	for {
		username, password, err := h.credentials()
		if err != nil {
			return fmt.Errorf("resources: http: authenticated: %w", err)
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.authUrl, nil)
		if err != nil {
			return fmt.Errorf("resources: http: authenticate: %w", err)
		}

		request.Header.Set("Connection", "Keep-Alive")
		request.Header.Set("User-Agent", UserAgent)
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

func (h *_httpWithAuth) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	for {
		r, err := h.HttpResources.Get(ctx, uri)
		if err == nil {
			return r, nil
		}

		if !errors.Is(err, ErrNotAuthenticated) {
			return nil, err
		}

		if err := h.Authenticate(ctx); err != nil {
			return nil, err
		}
	}
}

func WithAuthentication(r HttpResources, credentials CredentialsFunc, authUrl string) Resources {
	if credentials == nil {
		panic(fmt.Errorf("credentials func cannot be nil"))
	}
	return &_httpWithAuth{r, credentials, authUrl}
}

type _withRoot struct {
	Resources
	root string
}

func (r *_withRoot) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return r.Resources.Get(ctx, filepath.Join(r.root, path))
}

func WithRoot(r Resources, root string) Resources {
	return &_withRoot{r, root}
}

type _withUrl struct {
	Resources
	base string
}

func (r *_withUrl) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	uri, err := url.JoinPath(r.base, filepath.ToSlash(path))
	if err != nil {
		return nil, fmt.Errorf("resources: %w", err)
	}
	return r.Resources.Get(ctx, uri)
}

func WithUrl(r Resources, base string) Resources {
	return &_withUrl{r, base}
}
