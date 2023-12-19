package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/ocr"
)

var (
	buildInfo = "NO INFO"
	buildTag  = "NO TAG"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (Version=%s,%s) :\n", os.Args[0], buildTag, buildInfo)
		flag.PrintDefaults()
	}

	args, err := cfg.ParseFlags()
	if err != nil {
		slog.Error(err.Error())
		flag.Usage()
		os.Exit(1)
	}
	defer func() {
		if err := args.DB.Close(); err != nil {
			slog.Error("db close", "err", err)
		}
	}()

	err = ocr.Parse(ctx, args)
	if err != nil {
		slog.Error(err.Error())
	}
}
