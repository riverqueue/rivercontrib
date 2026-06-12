package otelriver_test

import (
	"log/slog"
	"os"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

func ExampleMiddleware() {
	middleware := otelriver.NewMiddleware(nil)

	_, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Hooks: []rivertype.Hook{
			// Install the OpenTelemetry middleware as a hook to emit producer
			// metrics from this River client.
			middleware,
		},
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn, ReplaceAttr: slogutil.NoLevelTime})),
		Middleware: []rivertype.Middleware{
			// Install the OpenTelemetry middleware to run for all jobs inserted
			// or worked by this River client.
			middleware,
		},
		TestOnly: true, // suitable only for use in tests; remove for live environments
	})
	if err != nil {
		panic(err)
	}

	// Output:
}
