package db

import (
	"slices"

	"github.com/danlock/pkg/errors"
	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/array"
)

func Setup(dbPath string) (*sqlite3.Conn, error) {
	db, err := sqlite3.Open(dbPath)
	if err != nil {
		return nil, errors.Errorf("sqlite3.Open %w", err)
	}
	array.Register(db)

	// the images table contains the path, our parsed text, and a hash of the image.
	// image_hash is prepended with the hash algorithm (md5:, blake2b:, etc...) to support upgrading the hash later.
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS images
		(path TEXT PRIMARY KEY, image_text TEXT NOT NULL, image_hash TEXT NOT NULL) STRICT`)
	if err != nil {
		return nil, errors.Errorf("db.Exec %w", err)
	}

	return db, nil
}

func FilterParsedImages(db *sqlite3.Conn, images []string) ([]string, error) {
	// TODO: For now we use the path to identify images. Eventually incorporate the hash to recognize an image after renames.
	stmt, _, err := db.Prepare(`
		SELECT path FROM images
		WHERE path IN array(?)
	`)
	if err != nil {
		return images, errors.Errorf("DB.Prepare path query %w", err)
	}

	err = stmt.BindPointer(1, sqlite3.Pointer(images))
	if err != nil {
		return images, errors.Errorf("stmt.BindPointer path query %w", err)
	}

	var parsedImages map[string]struct{}

	for stmt.Step() {
		parsedImages[stmt.ColumnText(0)] = struct{}{}
	}
	if err = stmt.Err(); err != nil {
		return images, errors.Errorf("stmt.Err path query %w", err)
	}

	return slices.DeleteFunc(images, func(s string) bool {
		_, wasParsed := parsedImages[s]
		return wasParsed
	}), nil
}

func InsertParsedText(db *sqlite3.Conn, path, text, hash string) error {
	stmt, _, err := db.Prepare(`
		INSERT INTO images (path, image_text, image_hash) VALUES (?,?,?)
	`)
	if err != nil {
		return errors.Errorf("DB.Prepare insert image query %w", err)
	}
	stmt.BindText(0, path)
	stmt.BindText(1, text)
	stmt.BindText(2, hash)
	return errors.Wrap(stmt.Exec())
}
