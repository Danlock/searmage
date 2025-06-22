package errors

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// DefaultFileSlogKey is the default slog.Attr key used for file:line information when an error is printed via log/slog.
// If DefaultFileSlogKey is set to "", file:line metadata will not be included in errors.
var DefaultFileSlogKey = "file"

// metaError is a structured stdlib Go error using slog.Attr for metadata.
// If printed with %+v it will also include the metadata, but by default only the error message is shown.
// It will also include the file:line information from the first error in the chain under the DefaultFileSlogKey.
// Meant for use with log/slog where everything converts to a slog.GroupValue when logged.
type metaError struct {
	error
	meta []slog.Attr
}

func (e metaError) Unwrap() error  { return e.error }
func (e metaError) String() string { return e.Error() }

// LogValue logs the error with the file:line information and any existing metadata.
func (e metaError) LogValue() slog.Value {
	return slog.GroupValue(append(
		UnwrapMeta(e), slog.String("msg", e.Error()))...)
}

func stringifyAttr(meta []slog.Attr) string {
	if len(meta) == 0 {
		return ""
	}

	var all strings.Builder
	all.WriteString("{")
	for i, attr := range meta {
		all.WriteString(attr.String())
		if i < len(meta)-1 {
			all.WriteString(",")
		}
	}
	all.WriteString("}")
	return all.String()
}

// Not sure how I feel about this. I like being able to print all at the metadata in a quick and dirty way
// but if a logger defaults to %+v it would annoyingly duplicate the metadata.
// However as slog is in the stdlib, it's fair to expect other loggers to conform to slog.LogValuer eventually.
func (e metaError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// This outputs all metadata for ease of debugging but really... just use log/slog.
			_, _ = io.WriteString(s, e.Error()+" "+stringifyAttr(UnwrapMeta(e)))
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	}
}

// UnwrapMeta pulls metadata from every error in the chain for structured logging purposes.
// Errors in this package implement slog.LogValuer and automatically include the metadata when used with slog.Log.
// This function is mainly exposed for use with other loggers that don't support structured logging from the stdlib.
func UnwrapMeta(err error) (meta []slog.Attr) {
	var merr metaError
	for errors.As(err, &merr) {
		meta = append(meta, merr.meta...)
		err = errors.Unwrap(merr)
	}
	return meta
}

// UnwrapMetaMap returns a map around an error's metadata.
// If the error lacks metadata an empty map is returned.
//
// Structured errors can be introspected and handled differently as needed.
// As this is a map, duplicate keys across the error chain are not allowed.
// If that is an issue for you, use UnwrapMeta instead.
//
// Seriously consider a sentinel error or custom error type before reaching for this.
// For example open source libraries would be better off publicly exposing custom error types for type safety.
//
// Using const keys is strongly recommended to avoid typos.
func UnwrapMetaMap(err error) map[string]slog.Value {
	meta := make(map[string]slog.Value)
	var merr metaError
	for errors.As(err, &merr) {
		for i := range merr.meta {
			meta[merr.meta[i].Key] = merr.meta[i].Value
		}
		err = errors.Unwrap(merr)
	}
	return meta
}
