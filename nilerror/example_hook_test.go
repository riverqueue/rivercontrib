package nilerror_test

import (
	"log/slog"

	"github.com/rivercontrib/nilerror"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
)

func ExampleHook() {
	_, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Hooks: []rivertype.Hook{
			// Install a nilerror check that will return an error when it
			// detects a nil struct wrapped in a non-nil error interface.
			nilerror.NewHook(nil),

			// Alternatively, suppress errors and produce warning logging.
			nilerror.NewHook(&nilerror.HookConfig{Suppress: true}),
		},
		Logger:   slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelWarn}),
		TestOnly: true, // suitable only for use in tests; remove for live environments
	})
	if err != nil {
		panic(err)
	}

	// Output:
}
