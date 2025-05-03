package otelriver_test

import (
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

func ExampleMiddleware() {
	_, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Logger: slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelWarn}),
		Middleware: []rivertype.Middleware{
			// Install the OpenTelemetry middleware to run for all jobs inserted
			// or worked by this River client.
			otelriver.NewMiddleware(nil),
		},
		TestOnly: true, // suitable only for use in tests; remove for live environments
	})
	if err != nil {
		panic(err)
	}

	// Output:
}
