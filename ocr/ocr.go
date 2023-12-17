package ocr

import (
	"context"
	_ "embed"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/danlock/gogosseract"
	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/cfg"
)

//go:embed eng.traineddata
var engTrainedData []byte

func Run(ctx context.Context, args cfg.Args) error {
	gogoPoolCfg := gogosseract.PoolConfig{}
	if args.TrainedData != nil {
		gogoPoolCfg.Config.TrainingData = args.TrainedData
	} else {
		gogoPoolCfg.TrainingDataBytes = engTrainedData
	}

	ocr, err := gogosseract.NewPool(ctx, 10, gogoPoolCfg)
	if err != nil {
		return errors.Wrap(err)
	}

	var wg sync.WaitGroup

	err = filepath.WalkDir(args.ImageDir, func(fPath string, d fs.DirEntry, err error) error {
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

		img, err := os.Open(fPath)
		if err != nil {
			return errors.Errorf("filepath.WalkDir os.Open %w", err)
		}
		wg.Add(1)
		// TODO: instead of spinning up N goroutines, send the path into a buffered channel or something
		go func() {
			text, err := ocr.ParseImage(ctx, img, gogosseract.ParseImageOptions{})
			slog.Info("gogosseract ParseImage", "path", fPath, "text", text, "err", err)
			wg.Done()
		}()

		return nil
	})
	if err != nil {
		return errors.Errorf("filepath.WalkDir %w", err)
	}

	wg.Wait()
	return nil
}
