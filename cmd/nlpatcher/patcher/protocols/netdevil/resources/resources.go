package resources

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Func = func(path string) (io.ReadCloser, error)

type Resources interface {
	Get(path string) (io.ReadCloser, error)
}

func FileSchemePath(uri string) string {
	return filepath.FromSlash(strings.TrimPrefix(uri, "/"))
}

func File(path string) (io.ReadCloser, error) {
	file, err := os.Open(filepath.FromSlash(path))
	if err != nil {
		return nil, fmt.Errorf("fetch file: %w", err)
	}
	return file, nil
}

func WithFileBase(f Func, root string) Func {
	return func(path string) (io.ReadCloser, error) {
		return f(filepath.Join(root, path))
	}
}

func Http(client *http.Client) Func {
	return func(uri string) (io.ReadCloser, error) {
		request, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch http: %w", err)
		}

		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("fetch http: %w", err)
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return response.Body, nil
		}

		// Allows the client to reuse the connection when using Keep-Alive
		io.Copy(io.Discard, response.Body)
		response.Body.Close()

		return nil, fmt.Errorf("fetch http: unhandled status: %s", response.Status)
	}
}

func WithUrlBase(f Func, base string) Func {
	return func(path string) (io.ReadCloser, error) {
		uri, err := url.JoinPath(base, filepath.ToSlash(path))
		if err != nil {
			return nil, fmt.Errorf("fetch http: %w", err)
		}

		return f(uri)
	}
}
