package datadogriver_test

import (
	"log/slog"
	"os"

	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"go.opentelemetry.io/otel"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

func Example_globalProvider() {
	provider := ddotel.NewTracerProvider()
	defer func() { _ = provider.Shutdown() }()
	otel.SetTracerProvider(provider)

	_, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn, ReplaceAttr: slogutil.NoLevelTime})),
		Middleware: []rivertype.Middleware{
			// Install the OpenTelemetry middleware to run for all jobs inserted
			// or worked by this River client. The global provider is used.
			otelriver.NewMiddleware(nil),
		},
		TestOnly: true, // suitable only for use in tests; remove for live environments
	})
	if err != nil {
		panic(err)
	}

	// Output:
}
