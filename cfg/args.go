package cfg

import (
	"database/sql"
	"flag"
	"os"
	"path"
	"runtime"

	"github.com/danlock/pkg/errors"
)

type Args struct {
	ImageDir string
	Search   string
	Workers  uint
	Debug    bool
	Clear    bool
	IsRegex  bool

	DB     *sql.DB
	DBPath string

	TrainedData     *os.File
	trainedDataPath string
}

func ParseFlags() (Args, error) {
	var a Args
	flag.UintVar(&a.Workers, "workers", uint(runtime.NumCPU()/3), "Number of workers used for parsing. More workers mean more CPU usage.")
	flag.StringVar(&a.ImageDir, "dir", "", "Path of an directory containing JPEG or PNG images to parse.")
	flag.StringVar(&a.trainedDataPath, "trained-data", "", "English training data is used by default, however other language data can be downloaded here (https://github.com/tesseract-ocr/tessdata_fast)")
	flag.StringVar(&a.DBPath, "db", path.Join(os.TempDir(), "searmage.sqlite3"), "Path to place the database where searmage indexes image text. Defaults to the temp directory.")
	flag.BoolVar(&a.Debug, "debug", false, "Enable debug logging.")
	flag.BoolVar(&a.Clear, "clear", false, "If set, clears the given database instead of parsing images.")
	flag.StringVar(&a.Search, "search", "", "If set, searches for the given text within previously parsed images instead of parsing images. (by default uses MATCH from https://www.sqlite.org/fts5.html)")
	flag.BoolVar(&a.IsRegex, "regex", false, "If set, -search is evaluated as REGEXP instead of MATCH using https://pkg.go.dev/regexp/syntax")
	flag.Parse()
	// short circuit if we aren't parsing images
	if a.Clear || a.Search != "" {
		return a, nil
	}

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

	return a, nil
}
