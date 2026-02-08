package origin

import (
	"context"
	"encoding/json"
	"encoding/xml"
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

type Resources interface {
	Get(ctx context.Context, path string) (io.ReadCloser, error)
}

func NewResources(uri string) (Resources, string, error) {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, "", err
	}

	switch url.Scheme {
	case "file":
		return &FS{}, filepath.FromSlash(strings.TrimPrefix(url.Path, "/")), nil
	case "http", "https":
		return &Http{Client: http.DefaultClient}, uri, nil
	default:
		return nil, "", fmt.Errorf("unknown scheme: %s", url.Scheme)
	}
}

type FS struct{}

func (FS) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.Clean(filepath.FromSlash(path)))
	if err != nil {
		return nil, fmt.Errorf("origin: fs: %w", err)
	}

	select {
	case <-ctx.Done():
		file.Close()
		return nil, ctx.Err()
	default:
		return file, nil
	}
}

var (
	ErrNotAuthenticated = errors.New("not authenticated")

	userAgent = "NimbusLauncher/" + version.Get().Name()
)

type Http struct {
	Client *http.Client
}

func (h Http) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("origin: http: %w", err)
	}

	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("User-Agent", userAgent)

	response, err := h.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("origin: http: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response.Body, nil
	}

	// Allows the client to reuse the connection when using Keep-Alive
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("origin: http: %w", ErrNotAuthenticated)
	}

	return nil, fmt.Errorf("origin: http: Get %s: unhandled status: %s", uri, response.Status)
}

type ResponseMessage struct {
	Message string `json:"error" xml:"error"`
}

type CredentialsFunc = func(authMessage string) (username string, password []byte, err error)

type HttpWithAuth struct {
	*Http

	credentials CredentialsFunc
	authUrl     string
}

func (h HttpWithAuth) extractResponseMessage(response *http.Response) string {
	message := struct {
		Message string `json:"error" xml:"error"`
	}{}

	switch response.Header.Get("Content-Type") {
	case "application/json":
		json.NewDecoder(response.Body).Decode(&message)
	case "application/xml", "text/xml":
		xml.NewDecoder(response.Body).Decode(&message)
	case "text/plain", "":
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return response.Status
		}
		return string(data)
	}

	if len(message.Message) == 0 {
		return response.Status
	}
	return message.Message
}

func (h HttpWithAuth) Authenticate(ctx context.Context) error {
	var lastMessage string
	for {
		username, password, err := h.credentials(lastMessage)
		if err != nil {
			return fmt.Errorf("origin: http with auth: %w", err)
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.authUrl, nil)
		if err != nil {
			return fmt.Errorf("origin: http with auth: %w", err)
		}

		request.Header.Set("Connection", "Keep-Alive")
		request.Header.Set("User-Agent", userAgent)
		request.SetBasicAuth(username, string(password))

		response, err := h.Client.Do(request)
		if err != nil {
			return fmt.Errorf("origin: http with auth: %w", err)
		}
		defer response.Body.Close()

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return nil
		}

		if response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden {
			return fmt.Errorf("origin: http with auth: unhandled status: %s", response.Status)
		}

		lastMessage = h.extractResponseMessage(response)
	}
}

func (h HttpWithAuth) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	for {
		r, err := h.Http.Get(ctx, uri)
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

func WithAuthentication(h *Http, credentials CredentialsFunc, authUrl string) *HttpWithAuth {
	if credentials == nil {
		panic(fmt.Errorf("credentials func cannot be nil"))
	}
	return &HttpWithAuth{h, credentials, authUrl}
}

type withRoot struct {
	Resources
	root string
}

func (r withRoot) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return r.Resources.Get(ctx, filepath.Join(r.root, filepath.Clean(path)))
}

func WithRoot(r Resources, root string) Resources {
	return &withRoot{r, root}
}

type withUrl struct {
	Resources
	base string
}

func (r withUrl) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	uri, err := url.JoinPath(r.base, filepath.ToSlash(path))
	if err != nil {
		return nil, fmt.Errorf("origin: with url: %v", err)
	}
	return r.Resources.Get(ctx, uri)
}

func WithUrl(r Resources, base string) Resources {
	return &withUrl{r, base}
}
