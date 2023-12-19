package cfg

import (
	"flag"
	"os"
	"path"

	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/db"
	"github.com/ncruces/go-sqlite3"
)

type Args struct {
	ImageDir string

	DB     *sqlite3.Conn
	dbPath string

	TrainedData     *os.File
	trainedDataPath string
}

func ParseFlags() (Args, error) {
	var a Args
	flag.StringVar(&a.ImageDir, "dir", "", "Path of an directory containing JPEG or PNG images")
	flag.StringVar(&a.trainedDataPath, "trained-data", "", "English training data is used by default, however other language data can be downloaded here (https://github.com/tesseract-ocr/tessdata_fast)")
	flag.StringVar(&a.dbPath, "db", path.Join(os.TempDir(), "searmage.sqlite3"), "Path to place the database where searmage indexes image text. Defaults to the temp directory.")
	flag.Parse()

	var err error

	if a.ImageDir == "" {
		return a, errors.New("-dir required")
	}

	if a.trainedDataPath != "" {
		a.TrainedData, err = os.Open(a.trainedDataPath)
		if err != nil {
			return a, errors.Errorf("-trained-data os.Open %w", err)
		}
	}

	a.DB, err = db.Setup(a.dbPath)
	if err != nil {
		return a, errors.Wrap(err)
	}

	return a, nil
}
