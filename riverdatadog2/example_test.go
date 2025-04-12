package riverdatadog_test

import (
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/datadogriver"
)

func Example_datadogMiddleware() {
	middleware := datadogriver.NewMiddleware(nil)

	_, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		JobInsertMiddleware: []rivertype.JobInsertMiddleware{
			middleware,
		},
		Logger:   slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelWarn}),
		TestOnly: true, // suitable only for use in tests; remove for live environments
		WorkerMiddleware: []rivertype.WorkerMiddleware{
			middleware,
		},
	})
	if err != nil {
		panic(err)
	}

	// Output:
}
