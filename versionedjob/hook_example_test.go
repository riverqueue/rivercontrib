package versionedjob_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdbtest"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivershared/util/testutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/versionedjob"
)

// VersionedJobArgsV1 is the V1 version of the versioned job args. It's present
// in this example for reference, but in real example it'd be removed in favor
// of only the latest version and version transformer.
//
// Initial version of the job. Contains only a name field. There was no version
// field because its author didn't yet know they were going to need successive
// versions.
type VersionedJobArgsV1 struct {
	Name string `json:"name"`
}

func (VersionedJobArgsV1) Kind() string { return (VersionedJobArgs{}).Kind() }

// VersionedJobArgsV2 is the V2 version of the versioned job args. It's present
// in this example for reference, but in real example it'd be removed in favor
// of only the latest version and version transformer.
//
// In V2, the name field was renamed to title. This is a direct mapping so the
// transformer just needs to move the value from one place to the other. A
// version field is added so we can differentiate V1 and V2.
type VersionedJobArgsV2 struct {
	Title   string `json:"title"`
	Version int    `json:"version"`
}

func (VersionedJobArgsV2) Kind() string { return (VersionedJobArgs{}).Kind() }

// VersionedJobArgs is the V3 (current) version of the versioned job args.
//
// In V3, a description field is added. When versioning forward, a default value
// can be derived from the title.
type VersionedJobArgs struct {
	Description string `json:"description"`
	Title       string `json:"title"`
	Version     int    `json:"version"`
}

func (VersionedJobArgs) Kind() string { return "versioned_job" }

//
// Worker
//

type VersionedJobWorker struct {
	river.WorkerDefaults[VersionedJobArgs]
}

func (w *VersionedJobWorker) Work(ctx context.Context, job *river.Job[VersionedJobArgs]) error {
	fmt.Printf("Job title: %s; description: %s\n", job.Args.Title, job.Args.Description)
	return nil
}

type VersionedJobTransformer struct{}

func (*VersionedJobTransformer) Kind() string { return (VersionedJobArgs{}).Kind() }

func (*VersionedJobTransformer) VersionTransform(ctx context.Context, job *rivertype.JobRow) error {
	// Extract version from job, defaulting to 1 if not present because we
	// assume that was before versioning was introduced.
	version := cmp.Or(gjson.GetBytes(job.EncodedArgs, "version").Int(), 1)

	var err error

	//
	// Here, we walk through each successive version, applying transformations
	// to bring it to its next version. If a job is multiple versions behind,
	// version transformations are one-by-one applied in order until the job's
	// args are fully modernized.
	//

	// Version change: V1 --> V2
	if version < 2 {
		version = 2

		job.EncodedArgs, err = sjson.SetBytes(job.EncodedArgs, "title", gjson.GetBytes(job.EncodedArgs, "name").String())
		if err != nil {
			return err
		}

		job.EncodedArgs, err = sjson.DeleteBytes(job.EncodedArgs, "name")
		if err != nil {
			return err
		}
	}

	// Version change: V2 --> V3
	if version < 3 {
		version = 3

		title := gjson.GetBytes(job.EncodedArgs, "title").String()
		if title == "" {
			return errors.New("no title found in job args")
		}

		job.EncodedArgs, err = sjson.SetBytes(job.EncodedArgs, "description", "A description of a "+title+".")
		if err != nil {
			return err
		}
	}

	// Not strictly necessary, but set version to latest.
	job.EncodedArgs, err = sjson.SetBytes(job.EncodedArgs, "version", version)
	if err != nil {
		return err
	}

	return nil
}

func ExampleHook() {
	ctx := context.Background()

	dbPool, err := pgxpool.New(ctx, riversharedtest.TestDatabaseURL())
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &VersionedJobWorker{})

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Hooks: []rivertype.Hook{
			versionedjob.NewHook(&versionedjob.HookConfig{
				Transformers: []versionedjob.VersionTransformer{
					&VersionedJobTransformer{},
				},
			}),
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

	// Insert all three versions of the job. In each case, thanks to the version
	// transformer, the worker will receive the same args and produce the same
	// output for each regardless of what version of the job went in.
	if _, err = riverClient.InsertMany(ctx, []river.InsertManyParams{
		{
			Args: VersionedJobArgsV1{
				Name: "My Job",
			},
		},
		{
			Args: VersionedJobArgsV2{
				Title:   "My Job",
				Version: 2,
			},
		},
		{
			Args: VersionedJobArgs{
				Title:       "My Job",
				Description: "A description of a My Job.",
				Version:     3,
			},
		},
	}); err != nil {
		panic(err)
	}

	if err := riverClient.Start(ctx); err != nil {
		panic(err)
	}

	// Wait for jobs to complete. Only needed for purposes of the example test.
	riversharedtest.WaitOrTimeoutN(testutil.PanicTB(), subscribeChan, 3)

	if err := riverClient.Stop(ctx); err != nil {
		panic(err)
	}

	// Output:
	// Job title: My Job; description: A description of a My Job.
	// Job title: My Job; description: A description of a My Job.
	// Job title: My Job; description: A description of a My Job.
}
