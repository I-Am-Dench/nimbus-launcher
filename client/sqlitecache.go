package client

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	CREATE_CLIENT_RESOURCES = "CREATE TABLE IF NOT EXISTS client_resources (path TEXT UNIQUE, mod_time BIGINT, data BLOB)"
	INSERT_CLIENT_RESOURCE  = "INSERT INTO client_resources VALUES (?, ?, ?)"
	QUERY_ALL_RESOURCES     = "SELECT * FROM client_resources"
	QUERY_RESOURCE          = "SELECT * FROM client_resources WHERE path = ?"
)

type sqliteCache struct {
	db      *sql.DB
	context context.Context
}

func (cache *sqliteCache) Execute(query string, args ...any) (sql.Result, error) {
	return cache.db.ExecContext(cache.context, query, args...)
}

func (cache *sqliteCache) Query(query string, args ...any) (*sql.Rows, error) {
	return cache.db.QueryContext(cache.context, query, args...)
}

func (cache *sqliteCache) QueryRow(query string, args ...any) *sql.Row {
	return cache.db.QueryRowContext(cache.context, query, args...)
}

func NewSqliteCache(directory string) (Cache, error) {
	cache := new(sqliteCache)

	dsn := fmt.Sprintf("file:%s", filepath.Join(directory, "client_cache.sqlite"))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	cache.db = db
	cache.context = context.Background()

	db.SetMaxOpenConns(1)

	_, err = cache.Execute(CREATE_CLIENT_RESOURCES)
	if err != nil {
		return nil, fmt.Errorf("could not initialize client cache: %v", err)
	}

	return cache, nil
}

func (cache *sqliteCache) Add(resource Resource) error {
	_, err := cache.Execute(INSERT_CLIENT_RESOURCE, resource.Path, resource.ModTime, resource.Data)
	return err
}

func (cache *sqliteCache) Get(path string) (Resource, error) {
	resource := Resource{}
	row := cache.QueryRow(QUERY_RESOURCE, path)

	err := row.Scan(&resource.Path, &resource.ModTime, &resource.Data)
	if err != nil {
		return Resource{}, fmt.Errorf("clientcache: could not query resource: %v", err)
	}

	return resource, nil
}

func (cache *sqliteCache) GetResources() ([]Resource, error) {
	rows, err := cache.Query(QUERY_ALL_RESOURCES)
	if err != nil {
		return []Resource{}, err
	}

	resources := []Resource{}
	for rows.Next() {
		resource := Resource{}
		err := rows.Scan(&resource.Path, &resource.ModTime, &resource.Data)
		if err == nil {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func (cache *sqliteCache) Has(path string) bool {
	_, err := cache.Get(path)
	return err == nil
}

func (cache *sqliteCache) Close() error {
	return cache.db.Close()
}
