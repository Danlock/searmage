package ocr

import (
	"context"
	"crypto/md5"
	_ "embed"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/danlock/gogosseract"
	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/db"
)

//go:embed eng.traineddata
var engTrainedData []byte

func Parse(ctx context.Context, args cfg.Args) error {
	start := time.Now()

	images, err := GetImagePaths(args.ImageDir)
	if err != nil {
		return errors.Wrap(err)
	}

	images, err = db.FilterParsedImages(args.DB, images)
	if err != nil {
		return errors.Wrap(err)
	}

	if len(images) == 0 {
		slog.Info("All images in -dir have been parsed already", "-dir", args.ImageDir)
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	workers := uint(runtime.NumCPU())

	// Generate image files for the Tesseract workers.
	// By buffering the channel to the amount of workers in the pool,
	// we ensure we don't open more image files than needed.
	imgChan := make(chan *os.File, workers)
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

	gogoPoolCfg := gogosseract.PoolConfig{}
	if args.TrainedData != nil {
		gogoPoolCfg.Config.TrainingData = args.TrainedData
	} else {
		gogoPoolCfg.TrainingDataBytes = engTrainedData
	}

	ocr, err := gogosseract.NewPool(ctx, workers, gogoPoolCfg)
	if err != nil {
		return errors.Wrap(err)
	}
	defer ocr.Close()

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
			// Spin up a goroutine that hashes, parses the image and stores the result in the database.
			// Since imgChan is bounded to the CPU count, there should be more than that many running at the same time.
			go func() (err error) {
				defer img.Close()
				defer func() { errChan <- err }()

				hasher := md5.New()
				io.Copy(hasher, img)

				_, err = img.Seek(0, 0)
				if err != nil {
					return errors.Errorf("img.Seek %w", err)
				}

				text, err := ocr.ParseImage(ctx, img, gogosseract.ParseImageOptions{
					ProgressCB: func(i int32) {
						slog.Info("progress", "%", i, "path", img.Name())
					},
				})
				if err != nil {
					return errors.Wrap(err)
				}

				return errors.Wrap(db.InsertParsedText(args.DB, img.Name(), text, "md5:"+string(hasher.Sum([]byte{}))))
			}()
		}

		if parsedImages == len(images) {
			break
		}
	}

	slog.Info("Finished parsing", "images", parsedImages, "duration", time.Since(start))

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
