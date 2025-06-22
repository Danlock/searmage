// Package errors prefixes the calling functions name to errors for simpler, smaller traces.
// This package tries to split the difference between github.com/pkg/errors and Go stdlib errors,
// with first class support for log/slog.
package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"runtime"
)

// caller is the number of stack frames to skip when determining the caller's package.func.
const caller = 4

// New creates a new error with the package.func of it's caller prepended.
// It also includes the file and line info of it's caller.
func New(text string) error { return ErrorfWithSkip(caller, text) }

// Errorf is like fmt.Errorf with the "package.func" of it's caller prepended.
// It also includes the file and line info of it's caller.
func Errorf(format string, a ...any) error { return ErrorfWithSkip(caller, format, a...) }

// ErrorfWithSkip is like fmt.Errorf with the "package.func" of the desired caller prepended.
// It also includes the file and line info of it's caller.
func ErrorfWithSkip(skip int, format string, a ...any) error {
	frame := callerFunc(skip)
	var meta []slog.Attr
	if DefaultFileSlogKey != "" {
		meta = []slog.Attr{
			slog.String(DefaultFileSlogKey, fmt.Sprintf("%s:%d", frame.File, frame.Line))}
	}
	return metaError{meta: meta,
		error: fmt.Errorf(prependCaller(format, frame), a...)}
}

// WrapAndPass wraps a typical error func with Wrap and passes the value through unchanged.
func WrapAndPass[T any](val T, err error) (T, error) { return val, WrapfWithSkip(err, caller, "") }

// WrapfAndPass wraps a typical error func with Wrapf and passes the value through unchanged.
func WrapfAndPass[T any](val T, err error, format string, a ...any) (T, error) {
	return val, WrapfWithSkip(err, caller, format, a...)
}

// Wrap wraps an error with the caller's package.func prepended.
// Similar to github.com/pkg/errors.Wrap and unlike fmt.Errorf it returns nil if err is nil.
// If not wrapping an error from this Go package it also includes the file and line info of it's caller.
func Wrap(err error) error { return WrapfWithSkip(err, caller, "") }

// Wrapf wraps an error with the caller's package.func prepended.
// Similar to github.com/pkg/errors.Wrapf and unlike fmt.Errorf it returns nil if err is nil.
// If not wrapping an error from this Go package it also includes the file and line info of it's caller.
func Wrapf(err error, format string, a ...any) error {
	return WrapfWithSkip(err, caller, format, a...)
}

// WrapfWithSkip wraps an error with the caller's package.func prepended.
// Similar to github.com/pkg/errors.Wrapf and unlike fmt.Errorf it returns nil if err is nil.
// If not wrapping an error from this Go package it also includes the file and line info of it's caller.
// skip is the number of stack frames to skip before recording the function info from runtime.Callers.
func WrapfWithSkip(err error, skip int, format string, a ...any) error {
	if err == nil {
		return nil
	}
	frame := callerFunc(skip)
	var meta []slog.Attr
	if DefaultFileSlogKey != "" {
		if _, exist := Into[metaError](err); !exist {
			meta = []slog.Attr{
				slog.String(DefaultFileSlogKey, fmt.Sprintf("%s:%d", frame.File, frame.Line))}
		}
	}

	if format == "" {
		format = "%w"
	} else {
		format += " %w"
	}

	return metaError{meta: meta,
		error: fmt.Errorf(prependCaller(format, frame), append(a, err)...)}
}

func callerFunc(skip int) runtime.Frame {
	var pcs [1]uintptr
	if runtime.Callers(skip, pcs[:]) == 0 {
		return runtime.Frame{}
	}
	frames := runtime.CallersFrames(pcs[:])
	if frames == nil {
		return runtime.Frame{}
	}
	frame, _ := frames.Next()
	return frame
}

func prependCaller(text string, f runtime.Frame) string {
	if f.Function == "" {
		return text
	}
	// runtime.Frame.Function gives back something like github.com/danlock/pkg.funcName.
	// with just the package name and the func name, nested errors look more readable by default.
	// We also avoid the ugly giant stack trace cluttering logs and looking similar to panics.
	// Now that the file:line of the original error is also within the metadata,
	// trimming the fat makes errors easier to parse at a glance.
	_, fName := path.Split(f.Function)
	return fmt.Sprint(fName, " ", text)
}

// Into finds the first error in err's chain that matches target type T, and if so, returns it.
// Into is a type-safe alternative to As.
func Into[T error](err error) (val T, ok bool) {
	return val, errors.As(err, &val)
}

// Must is a generic helper, like template.Must, that wraps a call to a function returning (T, error)
// and panics if the error is non-nil.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

// The following simply call the stdlib so users don't need to include both errors packages.

// ErrUnsupported indicates that a requested operation cannot be performed, because it is unsupported
var ErrUnsupported = errors.ErrUnsupported

// As finds the first error in err's tree that matches target, and if one is found, sets target to that error value and returns true. Otherwise, it returns false.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is reports whether any error in err's tree matches target.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// Join returns an error that wraps the given errors.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's type contains an Unwrap method returning error. Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
