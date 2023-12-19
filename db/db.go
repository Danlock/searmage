package db

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/danlock/pkg/errors"
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/array"
	"github.com/ncruces/go-sqlite3/ext/unicode"
)

func Setup(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := driver.Open(dbPath, func(c *sqlite3.Conn) error {
		array.Register(c)
		unicode.Register(c)
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("sql.Open %w", err)
	}

	// the images table contains the path, our parsed text, and a hash of the image.
	// image_hash is prepended with the hash algorithm (md5:, blake2b:, etc...) to support upgrading the hash later.
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS images
		(path TEXT PRIMARY KEY, image_text TEXT NOT NULL, image_hash TEXT NOT NULL) STRICT`)
	if err != nil {
		return nil, errors.Errorf("db.Exec %w", err)
	}

	// config is a generic table intended for misc config, such as the wazero WASM compilation cache.
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS config
		(key TEXT PRIMARY KEY, value ANY NOT NULL) STRICT`)
	if err != nil {
		return nil, errors.Errorf("db.Exec %w", err)
	}

	return db, nil
}

func FilterParsedImages(ctx context.Context, db *sql.DB, images []string) ([]string, error) {
	// TODO: For now we use the path to identify images. Eventually incorporate the hash to recognize an image after renames.
	rows, err := db.QueryContext(ctx, `
		SELECT path FROM images WHERE path IN array(?)
	`, sqlite3.Pointer(images))
	if err != nil {
		return images, errors.Errorf("db.QueryContext %w", err)
	}
	defer rows.Close()

	parsedImages := make(map[string]struct{})

	for rows.Next() {
		var path string
		rows.Scan(&path)
		if err != nil {
			return images, errors.Errorf("rows.Scan %w", err)
		}
		parsedImages[path] = struct{}{}
	}

	return slices.DeleteFunc(images, func(s string) bool {
		_, wasParsed := parsedImages[s]
		return wasParsed
	}), nil
}

func InsertParsedText(ctx context.Context, db *sql.DB, path, text, hash string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO images (path, image_text, image_hash) VALUES (?,?,?)
	`, path, text, hash)
	return errors.Wrap(err)
}

func SearchParsedText(ctx context.Context, db *sql.DB, search string, isRegex bool) ([]string, error) {
	searchQ := "SELECT path FROM images WHERE image_text LIKE ?"
	if isRegex {
		searchQ = strings.Replace(searchQ, "LIKE", "REGEXP", 1)
	}

	rows, err := db.QueryContext(ctx, searchQ, search)
	if err != nil {
		return nil, errors.Errorf("db.QueryContext %w", err)
	}
	defer rows.Close()

	var matchingImages []string

	for rows.Next() {
		var path string
		rows.Scan(&path)
		if err != nil {
			return nil, errors.Errorf("rows.Scan %w", err)
		}
		matchingImages = append(matchingImages, path)
	}

	return matchingImages, nil
}
