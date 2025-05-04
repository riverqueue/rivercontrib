// Package nilerror provides a River hook for detecting a common Go error where
// a nil struct value is wrapped in a non-nil interface value. This commonly
// causes trouble with the error interface, where an unintentional nil error is
// returned.
//
// See: https://go.dev/doc/faq#nil_error.
//
// The package must use reflection to work, and therefore necessitates some
// overhead from doing so. Its recommended use is to only the hook in test
// environments and detect non-nil nil errors via thorough testing, but it can
// only be used in production environments and configured to return an error on
// a detected problem or log a warning.
package nilerror

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivertype"
)

// HookConfig is configuration for the nilerror hook.
type HookConfig struct {
	// Suppress causes the hook to suppress detected nil struct values wrapped
	// in non-nil error interface values and produce warning logging instead.
	Suppress bool
}

// Hook is a River hook that detects nil error structs accidentally wrapped in a
// non-nil error interface, and either returns an error or logs a warning.
//
// See: https://go.dev/doc/faq#nil_error.
type Hook struct {
	baseservice.BaseService
	rivertype.Hook
	config *HookConfig
}

// NewHook initializes a new River nilerror hook.
//
// config may be nil.
func NewHook(config *HookConfig) *Hook {
	if config == nil {
		config = &HookConfig{}
	}
	return &Hook{config: config}
}

func (h *Hook) WorkEnd(ctx context.Context, err error) error {
	if err != nil {
		errVal := reflect.ValueOf(err)
		if errVal.IsNil() {
			var (
				nonPtrType  = errVal.Type().Elem()
				packagePath = nonPtrType.PkgPath()
				lastSlash   = strings.LastIndex(packagePath, "/")
				packageName = packagePath[lastSlash+1:]
				nilPtrName  = fmt.Sprintf("(*%s.%s)(<nil>)", packageName, nonPtrType.Name())
				message     = "non-nil error containing nil internal value (see: https://go.dev/doc/faq#nil_error); probably a bug: " + nilPtrName
			)

			if h.config.Suppress {
				h.Logger.WarnContext(ctx, h.Name+": Got "+message)
				return nil
			}

			return errors.New(message)
		}
	}

	return err
}
