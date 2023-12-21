//go:build cgo

package ocr

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"io"
	"os"

	"github.com/danlock/pkg/errors"
	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/db"
	"github.com/otiai10/gosseract/v2"
)

// setupWorkers creates a pool of gosseract workers, and returns a worker function that parses the image, stores the result in sqlite and returns an error or nil to errChan.
func setupWorkers(ctx context.Context, args cfg.Args) (WorkerFunc, error) {
	return func(ctx context.Context, errChan chan<- error, img *os.File) (err error) {
		tess := gosseract.NewClient()
		defer tess.Close()

		defer img.Close()
		defer func() { errChan <- err }()

		hasher := md5.New()
		io.Copy(hasher, img)
		hash := "md5:" + base64.RawURLEncoding.EncodeToString(hasher.Sum([]byte{}))
		err = tess.SetImage(img.Name())
		if err != nil {
			return errors.Errorf("tess.SetImage %w", err)
		}

		text, err := tess.Text()
		if err != nil {
			return errors.Wrap(err)
		}

		return errors.Wrap(db.InsertParsedText(ctx, args.DB, img.Name(), text, hash))
	}, nil

}
