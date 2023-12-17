// Personalize the errors stdlib to prepend the calling functions name to errors for simple traces
// An example of how this work can be seen at github.com/danlock/gogosseract.
// Example error message from gogosseract:
// gogosseract.NewPool failed worker setup due to gogosseract.(*Pool).runTesseract gogosseract.New gogosseract.Tesseract.createByteView wasm.GetReaderSize io.Reader was empty
package errors

import (
	"errors"
	"fmt"
	"path"
	"runtime"
)

// New creates a new error with the package.func of it's caller prepended.
func New(text string) error {
	return errors.New(prependCaller(text, 2))
}

// Errorf is like fmt.Errorf with the "package.func" of it's caller prepended.
func Errorf(format string, a ...any) error {
	return fmt.Errorf(prependCaller(format, 2), a...)
}

// Errorf is like fmt.Errorf with the "package.func" of the desired caller prepended.
func ErrorfWithSkip(format string, skip int, a ...any) error {
	return fmt.Errorf(prependCaller(format, skip), a...)
}

// Wrap wraps an error with the caller's package.func prepended.
// Similar to github.com/pkg/errors.Wrap it also returns nil if err is nil, unlike fmt.Errorf.
// Exclusively for wrapping an error with nothing more than the calling functions name, as more involved errors
// should use Errorf to match up a tiny bit closer with the Go stdlib.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(prependCaller("%w", 2), err)
}

func prependCaller(text string, skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	// f.Name() gives back something like github.com/danlock/pkg.funcName.
	// with just the package name and the func name, nested errors look more readable by default.
	// We also avoid an ugly giant stack trace that won't always get printed out.
	_, fName := path.Split(f.Name())
	return fmt.Sprint(fName, " ", text)
}

// The following simply call the stdlib so users don't need to include both errors packages.

var ErrUnsupported = errors.ErrUnsupported

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}
