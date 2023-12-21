//go:build !cgo

package ocr

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"io"
	"log/slog"
	"os"

	"github.com/danlock/gogosseract"
	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/db"
)

// setupWorkers creates a pool of gogosseract workers, and returns a worker function that parses the image, stores the result in sqlite and returns an error or nil to errChan.
func setupWorkers(ctx context.Context, args cfg.Args) (WorkerFunc, error) {
	gogoPoolCfg := gogosseract.PoolConfig{}
	if args.TrainedData != nil {
		gogoPoolCfg.Config.TrainingData = args.TrainedData
	} else {
		gogoPoolCfg.TrainingDataBytes = engTrainedData
	}

	ocr, err := gogosseract.NewPool(ctx, args.Workers, gogoPoolCfg)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	context.AfterFunc(ctx, ocr.Close)

	return func(ctx context.Context, errChan chan<- error, img *os.File) (err error) {
		defer img.Close()
		defer func() { errChan <- err }()

		hasher := md5.New()
		io.Copy(hasher, img)
		hash := "md5:" + base64.RawURLEncoding.EncodeToString(hasher.Sum([]byte{}))

		_, err = img.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Errorf("img.Seek %w", err)
		}

		text, err := ocr.ParseImage(ctx, img, gogosseract.ParseImageOptions{
			ProgressCB: func(i int32) {
				slog.Debug("progress", "%", i, "path", img.Name())
			},
		})
		if err != nil {
			return errors.Wrap(err)
		}

		return errors.Wrap(db.InsertParsedText(ctx, args.DB, img.Name(), text, hash))
	}, nil

}
