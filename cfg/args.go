package cfg

import (
	"flag"
	"os"

	"github.com/danlock/pkg/errors"
)

type Args struct {
	ImageDir string

	TrainedData     *os.File
	trainedDataPath string
}

func ParseFlags() (Args, error) {
	var a Args
	flag.StringVar(&a.ImageDir, "dir", "", "Path of an directory containing JPEG or PNG images")
	flag.StringVar(&a.trainedDataPath, "trained-data", "", "English training data is used by default, Other language training data can be downloaded here (https://github.com/tesseract-ocr/tessdata_fast)")
	flag.Parse()

	var err error

	if a.trainedDataPath != "" {
		a.TrainedData, err = os.Open(a.trainedDataPath)
		if err != nil {
			return a, errors.Errorf("-trained-data os.Open %w", err)
		}
	}

	if a.ImageDir == "" {
		return a, errors.New("-dir required")
	}

	return a, nil
}
