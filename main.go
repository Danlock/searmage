package main

import (
	"errors"
	"log"
	"path/filepath"

	"github.com/danlock/regex-img/ocr"
	"github.com/spf13/cobra"
)

func main() {
	rootCLI := &cobra.Command{
		Use:   "regex-img",
		Short: "CLI for grepping your way through an directory of images",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return errors.New("imgpath can't be empty!")
			} else if filepath.Ext(args[0]) != ".jpg" &&
				filepath.Ext(args[0]) != ".jpeg" &&
				filepath.Ext(args[0]) != ".png" {
				return errors.New("Only JPG or PNG are accepted!")
			}
			ocrP, err := ocr.NewOCRParser(filepath.Clean(args[0]))
			defer ocrP.Close()
			if err != nil {
				return err
			}
			_, err = ocrP.ScanImages()
			return err
		},
	}

	if err := rootCLI.Execute(); err != nil {
		log.Fatalf("\n%+v", err)
	}
}
