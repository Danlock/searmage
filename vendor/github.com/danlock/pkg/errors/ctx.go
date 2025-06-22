package errors

import (
	"context"
	"fmt"
	"log/slog"
)

type ctxKey struct{}

// AddMetaToCtx adds metadata to the context that will be added to the error once WrapMetaCtx is called.
func AddMetaToCtx(ctx context.Context, meta ...slog.Attr) context.Context {
	if ctx == nil {
		return nil
	}
	parent, ok := ctx.Value(ctxKey{}).([]slog.Attr)
	if ok {
		meta = append(meta, parent...)
	}
	return context.WithValue(ctx, ctxKey{}, meta)
}

// WrapMetaCtx wraps an error with metadata for structured logging.
// Similar to github.com/pkg/errors.Wrap and unlike fmt.Errorf it returns nil if err is nil.
// If not wrapping an error from this Go package it also includes the file and line info of it's caller.
// Metadata from the ctx added via CtxWithMeta will also be added to the error, if the context is set.
func WrapMetaCtx(ctx context.Context, err error, meta ...slog.Attr) error {
	if err == nil {
		return nil
	}
	if DefaultFileSlogKey != "" {
		if _, exist := Into[metaError](err); !exist {
			frame := callerFunc(caller - 1)
			meta = append(meta, slog.String(DefaultFileSlogKey, fmt.Sprintf("%s:%d", frame.File, frame.Line)))
		}
	}
	if ctx != nil {
		parent, ok := ctx.Value(ctxKey{}).([]slog.Attr)
		if ok {
			meta = append(meta, parent...)
		}
	}
	return metaError{error: err, meta: meta}
}

// WrapMeta is like WrapMetaCtx without the context.
func WrapMeta(err error, meta ...slog.Attr) error {
	if err == nil {
		return nil
	}
	if DefaultFileSlogKey != "" {
		if _, exist := Into[metaError](err); !exist {
			frame := callerFunc(caller - 1)
			meta = append(meta, slog.String(DefaultFileSlogKey, fmt.Sprintf("%s:%d", frame.File, frame.Line)))
		}
	}
	return metaError{error: err, meta: meta}
}
