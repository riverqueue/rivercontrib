package panictoerror_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdbtest"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivershared/util/testutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/panictoerror"
)

type PanicErrorArgs struct{}

func (PanicErrorArgs) Kind() string { return "custom_error" }

type PanicErrorWorker struct {
	river.WorkerDefaults[PanicErrorArgs]
}

func (w *PanicErrorWorker) Work(ctx context.Context, job *river.Job[PanicErrorArgs]) error {
	panic("this worker always panics!")
}

func ExampleMiddleware() {
	ctx := context.Background()

	dbPool, err := pgxpool.New(ctx, riversharedtest.TestDatabaseURL())
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &PanicErrorWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn, ReplaceAttr: slogutil.NoLevelTime})),
		Middleware: []rivertype.Middleware{
			// Layer a middleware above panictoerror.Middleware that takes a
			// return error and prints it to stdout for the purpose of this test.
			river.WorkerMiddlewareFunc(func(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
				var panicErr *panictoerror.PanicError
				if err := doInner(ctx); errors.As(err, &panicErr) {
					fmt.Printf("error from doInner: %s", panicErr.Cause)
				}
				return nil
			}),

			// This middleware coverts the panic to an error.
			panictoerror.NewMiddleware(nil),
		},
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Schema:   riverdbtest.TestSchema(ctx, testutil.PanicTB(), riverpgxv5.New(dbPool), nil), // only necessary for the example test
		TestOnly: true,                                                                         // suitable only for use in tests; remove for live environments
		Workers:  workers,
	})
	if err != nil {
		panic(err)
	}

	// Out of example scope, but used to wait until a job is worked.
	subscribeChan, subscribeCancel := riverClient.Subscribe(river.EventKindJobCompleted)
	defer subscribeCancel()

	if _, err = riverClient.Insert(ctx, PanicErrorArgs{}, nil); err != nil {
		panic(err)
	}

	if err := riverClient.Start(ctx); err != nil {
		panic(err)
	}

	// Wait for jobs to complete. Only needed for purposes of the example test.
	riversharedtest.WaitOrTimeoutN(testutil.PanicTB(), subscribeChan, 1)

	if err := riverClient.Stop(ctx); err != nil {
		panic(err)
	}

	// Output:
	// error from doInner: this worker always panics!
}
