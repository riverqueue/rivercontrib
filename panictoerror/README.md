# panictoerror [![Build Status](https://github.com/riverqueue/rivercontrib/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/riverqueue/rivercontrib/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/riverqueue/rivercontrib.svg)](https://pkg.go.dev/github.com/riverqueue/rivercontrib/nilerror)

Provides a `rivertype.WorkerMiddleware` that recovers panics that may have occurred deeper in the middleware stack (i.e. an inner middleware or the worker itself), converts those panics to errors, and returns those errors up the stack. This may be convenient in some cases so that middleware further up the stack need only have one way to handle either return errors or panic values.

``` go
// A worker implementation which will always panic.
func (w *PanicErrorWorker) Work(ctx context.Context, job *river.Job[PanicErrorArgs]) error {
    panic("this worker always panics!")
}

riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
    Middleware: []rivertype.Middleware{
        // This middleware further up the stack always receives an error instead
        // of a panic because `panictoerror.Middleware` is nested below it.
        river.WorkerMiddlewareFunc(func(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
            if err := doInner(ctx); err != nil {
                panicErr := err.(*panictoerror.PanicError)
                fmt.Printf("error from doInner: %s", panicErr.Cause)
            }
            return nil
        }),

        // This middleware coverts the panic to an error.
        panictoerror.NewMiddleware(nil),
    },
}
```

Based [on work](https://github.com/riverqueue/river/issues/1073#issuecomment-3515520394) from [@jerbob92](https://github.com/jerbob92).