# versionedjob [![Build Status](https://github.com/riverqueue/rivercontrib/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/riverqueue/rivercontrib/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/riverqueue/rivercontrib.svg)](https://pkg.go.dev/github.com/riverqueue/rivercontrib/versionedjob)

Provides a River hook with a simple job versioning framework. **Version transformers** are written for versioned jobs containing procedures for upgrading jobs that were encoded as older versions to the most modern version. This allows for workers to be implemented as if all job versions will be the most modern version only, keeping code simpler.

```go
// VersionTransformer defines how to perform transformations between versions
// for a specific job kind.
type VersionTransformer interface {
    // Kind is the job kind that this transformer applies to.
    Kind() string

    // VersionTransform applies version transformations to the given job. Version
    // transformations are fully defined according to the user, as well as how a
    // version is extracted from the job's args.
    //
    // Generally, this function should extract a version from the job, then
    // apply versions one by one until it's fully modernized to the point where
    // it can be successfully run by its worker.
    VersionTransform(job *rivertype.JobRow) error
}
```

## Example

Below are three versions of the same job: `VersionedJobArgsV1`, `VersionedJobArgsV2`, and the current version, `VersionedJobArgs`. From V1 to V2, `name` was renamed to `title`, and a `version` field added to track version. In V3, a new `description` property was added. A real program would only keep the latest version (`VersionedJobArgs`), but this example shows all three for reference.

```go
type VersionedJobArgsV1 struct {
    Name string `json:"name"`
}

type VersionedJobArgsV2 struct {
    Title   string `json:"title"`
    Version int    `json:"version"`
}

type VersionedJobArgs struct {
    Description string `json:"description"`
    Title       string `json:"title"`
    Version     int    `json:"version"`
}
```

The worker for `VersionedJobArgs` is written so it only handles the latest version (`title` instead of `name` and assumes `description` is present). This is possible because a `VersionTransformer` will handle migrating jobs from old versions to new ones before they hit the worker.

```go
type VersionedJobWorker struct {
    river.WorkerDefaults[VersionedJobArgs]
}

func (w *VersionedJobWorker) Work(ctx context.Context, job *river.Job[VersionedJobArgs]) error {
    fmt.Printf("Job title: %s; description: %s\n", job.Args.Title, job.Args.Description)
    return nil
}
```

The `VersionTransformer` implementation handles version upgrades one by one. Jobs which are multiple versions old can still be upgraded because multiple version changes can be applied in one go. This implementation uses `gjson`/`sjson` so that each change need only know a minimum about the data object in question and that unknown fields are retained. Other approaches are possible though, including using only Go's built-in `gjson` package.

```go
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
```

A River client is initialized with the `versiondjob` hook and transformer installed:

```go
riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
    Hooks: []rivertype.Hook{
        versionedjob.NewHook(&versionedjob.HookConfig{
            Transformers: []versionedjob.VersionTransformer{
                &VersionedJobTransformer{},
            },
        }),
    },
})
if err != nil {
    panic(err)
}
```

With all that in place, a job of any version can be inserted and thanks to the version transformer modernizing the older ones, the worker will produce the same result regardless of input.

```go
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
```

```go
// Output:
// Job title: My Job; description: A description of a My Job.
// Job title: My Job; description: A description of a My Job.
// Job title: My Job; description: A description of a My Job.
```
