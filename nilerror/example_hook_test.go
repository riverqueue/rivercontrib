package nilerror_test

import (
	"context"
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
	"github.com/riverqueue/rivercontrib/nilerror"
)

type CustomError struct{}

func (*CustomError) Error() string {
	return "my custom error"
}

type CustomErrorArgs struct{}

func (CustomErrorArgs) Kind() string { return "custom_error" }

type CustomErrorWorker struct {
	river.WorkerDefaults[CustomErrorArgs]
}

func (w *CustomErrorWorker) Work(ctx context.Context, job *river.Job[CustomErrorArgs]) error {
	var customErr *CustomError // nil error, but non-nil when wrapped in an error interface
	return customErr
}

func ExampleHook() {
	ctx := context.Background()

	dbPool, err := pgxpool.New(ctx, riversharedtest.TestDatabaseURL())
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &CustomErrorWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Hooks: []rivertype.Hook{
			// Suppress option prevents errors in favor of warning logging when
			// a nil struct wrapped in a non-nil error interface is detected.
			nilerror.NewHook(&nilerror.HookConfig{Suppress: true}),

			// Alternatively, return an error and fail jobs instead.
			// nilerror.NewHook(nil),
		},
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn, ReplaceAttr: slogutil.NoLevelTime})),
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

	if _, err = riverClient.Insert(ctx, CustomErrorArgs{}, nil); err != nil {
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
	// msg="nilerror.Hook: Got non-nil error containing nil internal value (see: https://go.dev/doc/faq#nil_error); probably a bug: (*nilerror_test.CustomError)(<nil>)"
}
