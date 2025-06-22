package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/danlock/searmage/cfg"
	"github.com/danlock/searmage/db"
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
		slog.Error("config", "err", err)
		flag.Usage()
		os.Exit(1)
	}

	if args.Debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	if args.Clear {
		slog.Info("-clear was set, database gone...", "err", os.Remove(args.DBPath))
		return
	}

	args.DB, err = db.Setup(ctx, args.DBPath)
	if err != nil {
		slog.Error("sqlite", "err", err)
		flag.Usage()
		os.Exit(1)
	}
	defer func() {
		if err := args.DB.Close(); err != nil {
			slog.Error("db close", "err", err)
		}
	}()

	if args.Search != "" {
		images, err := db.SearchParsedText(ctx, args.DB, args.Search, args.IsRegex)
		slog.Info("-search was set, found the following...", "err", err, "images", images)
		return
	}

	err = ocr.Parse(ctx, args)
	if err != nil {
		slog.Error("ocr", "err", err)
	}
}
