package riverdatadog2

import (
	"context"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type MiddlewareConfig struct{}

type Middleware struct {
	river.JobInsertMiddlewareDefaults
	river.WorkerMiddlewareDefaults
	globalStartOpts []tracer.StartSpanOption
}

func NewMiddleware(config *MiddlewareConfig) *Middleware {
	globalOpts := []tracer.StartSpanOption{
		tracer.Measured(),
		tracer.Tag(ext.MessagingSystem, "river"),
	}

	return &Middleware{
		globalStartOpts: globalOpts,
	}
}

func (m *Middleware) InsertMany(ctx context.Context, manyParams []*rivertype.JobInsertParams, doInner func(ctx context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "river.insert_many", m.globalStartOpts...)

	var finishOpts []tracer.FinishOption
	defer func() { span.Finish(finishOpts...) }()

	insertRes, err := doInner(ctx)
	if err != nil {
		finishOpts = append(finishOpts, tracer.WithError(err))
	}
	return insertRes, nil
}

func (m *Middleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "river.work", m.globalStartOpts...)
	span.SetTag("attempt", job.Attempt)
	span.SetTag("kind", job.Kind)
	span.SetTag("queue", job.Queue)

	var finishOpts []tracer.FinishOption
	defer func() { span.Finish(finishOpts...) }()

	err := doInner(ctx)
	if err != nil {
		finishOpts = append(finishOpts, tracer.WithError(err))
	}
	return err
}
