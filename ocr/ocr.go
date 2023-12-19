package ocr

import (
	"context"
	_ "embed"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/db"
)

//go:embed eng.traineddata
var engTrainedData []byte

type WorkerFunc func(ctx context.Context, errChan chan<- error, img *os.File) error

func Parse(ctx context.Context, args cfg.Args) error {
	start := time.Now()

	images, err := GetImagePaths(args.ImageDir)
	if err != nil {
		return errors.Wrap(err)
	}

	images, err = db.FilterParsedImages(ctx, args.DB, images)
	if err != nil {
		return errors.Wrap(err)
	}

	filteredImageCount := len(images)

	if filteredImageCount == 0 {
		slog.Info("All images in -dir have been parsed already", "-dir", args.ImageDir)
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	slog.Info("searmage processing...", "count", filteredImageCount, "workers", args.Workers)

	// Generate image files for the Tesseract workers.
	// By buffering the channel to the amount of workers in the pool,
	// we ensure we don't open more image files than needed at a time.
	imgChan := make(chan *os.File, args.Workers)
	errChan := make(chan error, 1)
	go func() {
		for _, fPath := range images {
			img, err := os.Open(fPath)
			if err != nil {
				errChan <- errors.Errorf("os.Open %w", err)
				return
			}
			select {
			case <-ctx.Done():
				img.Close()
				return
			case imgChan <- img:
			}
		}
	}()

	process, err := setupWorkers(ctx, args)
	if err != nil {
		return errors.Wrap(err)
	}
	parsedImages := 0

	for {
		select {
		case <-ctx.Done():
			return errors.Errorf(" context done before  ", ctx.Err())
		case err := <-errChan:
			if err != nil {
				return errors.Wrap(err)
			}
			parsedImages++

		case img := <-imgChan:
			go process(ctx, errChan, img)
		}

		if parsedImages == len(images) {
			break
		}
	}

	slog.Info("Finished parsing", "count", parsedImages, "duration", time.Since(start))
	return nil
}

func GetImagePaths(dir string) ([]string, error) {
	var imagePaths []string
	err := filepath.WalkDir(dir, func(fPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Errorf("filepath.WalkDir os.Open %w", err)
		}

		if d.IsDir() {
			return nil
		}

		switch path.Ext(fPath) {
		case ".jpg", ".jpeg", ".png":
		default:
			return nil
		}

		imagePaths = append(imagePaths, fPath)
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("filepath.WalkDir %w", err)
	}
	return imagePaths, nil
}
