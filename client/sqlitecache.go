package client

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const (
	CREATE_REPLACED_RESOURCES    = "CREATE TABLE IF NOT EXISTS replaced_resources (path TEXT UNIQUE, mod_time BIGINT, data BLOB)"
	INSERT_REPLACED_RESOURCE     = "INSERT INTO replaced_resources VALUES (?, ?, ?)"
	QUERY_ALL_REPLACED_RESOURCES = "SELECT * FROM replaced_resources"
	QUERY_REPLACED_RESOURCE      = "SELECT * FROM replaced_resources WHERE path = ?"

	CREATE_ADDED_RESOURCES    = "CREATE TABLE IF NOT EXISTS added_resources (path TEXT UNIQUE PRIMARY KEY)"
	INSERT_ADDED_RESOURCE     = "INSERT INTO added_resources VALUES (?)"
	QUERY_ALL_ADDED_RESOURCES = "SELECT * FROM added_resources"
	QUERY_ADDED_RESOURCE      = "SELECT * FROM added_resources WHERE path = ?"
)

type sqliteBase struct {
	db  *sql.DB
	ctx context.Context
}

func (cache *sqliteBase) Execute(query string, args ...any) (sql.Result, error) {
	return cache.db.ExecContext(cache.ctx, query, args...)
}

func (cache *sqliteBase) Query(query string, args ...any) (*sql.Rows, error) {
	return cache.db.QueryContext(cache.ctx, query, args...)
}

func (cache *sqliteBase) QueryRow(query string, args ...any) *sql.Row {
	return cache.db.QueryRowContext(cache.ctx, query, args...)
}

type sqliteReplacements struct {
	sqliteBase
}

func (cache *sqliteReplacements) Add(resource Resource) error {
	_, err := cache.Execute(INSERT_REPLACED_RESOURCE, resource.Path, resource.ModTime, resource.Data)
	return err
}

func (cache *sqliteReplacements) Get(path string) (Resource, error) {
	resource := Resource{}
	row := cache.QueryRow(QUERY_REPLACED_RESOURCE, path)

	err := row.Scan(&resource.Path, &resource.ModTime, &resource.Data)
	if err != nil {
		return Resource{}, fmt.Errorf("sqlite replacement cache: could not query resource: %w", err)
	}

	return resource, nil
}

func (cache *sqliteReplacements) List() ([]Resource, error) {
	rows, err := cache.Query(QUERY_ALL_REPLACED_RESOURCES)
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

func (cache *sqliteReplacements) Has(path string) bool {
	_, err := cache.Get(path)
	return err == nil
}

type sqliteAdditions struct {
	sqliteBase
}

func (cache *sqliteAdditions) Add(path string) error {
	_, err := cache.Execute(INSERT_ADDED_RESOURCE, path)
	return err
}

// This function is only implemented to satisfy the Cache[string] interface.
func (cache *sqliteAdditions) Get(path string) (string, error) {
	var queriedPath string
	row := cache.QueryRow(QUERY_ADDED_RESOURCE, path)

	err := row.Scan(&queriedPath)
	if err != nil {
		return "", fmt.Errorf("sqlite additions cache: could not query path: %w", err)
	}

	return queriedPath, nil
}

func (cache *sqliteAdditions) List() ([]string, error) {
	rows, err := cache.Query(QUERY_ALL_ADDED_RESOURCES)
	if err != nil {
		return []string{}, err
	}

	paths := []string{}
	for rows.Next() {
		var path string
		err := rows.Scan(&path)
		if err == nil {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

func (cache *sqliteAdditions) Has(path string) bool {
	_, err := cache.Get(path)
	return err == nil
}

type sqliteResources struct {
	replacements sqliteReplacements
	additions    sqliteAdditions

	db     *sql.DB
	cancel context.CancelFunc
}

func NewSqliteResources(path string) (Resources, error) {
	dsn := fmt.Sprintf("file:%s", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithCancel(context.Background())

	replacements := sqliteReplacements{sqliteBase{db, ctx}}
	additions := sqliteAdditions{sqliteBase{db, ctx}}

	resources := &sqliteResources{
		replacements: replacements,
		additions:    additions,

		db:     db,
		cancel: cancel,
	}

	_, err = resources.db.ExecContext(ctx, CREATE_REPLACED_RESOURCES)
	if err != nil {
		resources.Close()
		return nil, fmt.Errorf("could not initialize sqlite replacements cache: %w", err)
	}

	_, err = resources.db.ExecContext(ctx, CREATE_ADDED_RESOURCES)
	if err != nil {
		resources.Close()
		return nil, fmt.Errorf("could not initialize sqlite additions cache: %w", err)
	}

	return resources, nil
}

func (resources *sqliteResources) Replacements() Cache[Resource] {
	return &resources.replacements
}

func (resources *sqliteResources) Additions() Cache[string] {
	return &resources.additions
}

func (resources *sqliteResources) Close() error {
	resources.cancel()
	return resources.db.Close()
}
