package otelriver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/riverqueue/river/rivertype"
)

// Verify interface compliance.
var (
	_ rivertype.JobInsertMiddleware = &Middleware{}
	_ rivertype.WorkerMiddleware    = &Middleware{}
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	type testBundle struct {
		exporter *tracetest.InMemoryExporter
	}

	setup := func(t *testing.T) (*Middleware, *testBundle) {
		t.Helper()

		exporter := tracetest.NewInMemoryExporter()

		return NewMiddleware(&MiddlewareConfig{
				TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter)),
			}), &testBundle{
				exporter: exporter,
			}
	}

	t.Run("InsertManySuccess", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			return []*rivertype.JobInsertResult{
				{Job: &rivertype.JobRow{ID: 123}},
			}, nil
		}

		insertRes, err := middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		require.NoError(t, err)
		require.Equal(t, []*rivertype.JobInsertResult{
			{Job: &rivertype.JobRow{ID: 123}},
		}, insertRes)

		spans := bundle.exporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "river.insert_many", span.Name)
		require.Equal(t, codes.Ok, span.Status.Code)
	})

	t.Run("InsertManyError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			return nil, errors.New("error from doInner")
		}

		_, err := middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		require.EqualError(t, err, "error from doInner")

		spans := bundle.exporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "river.insert_many", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "error from doInner", span.Status.Description)
	})

	// Make sure the middleware can fall back to a global provider.
	t.Run("InsertManyEmptyConfig", func(t *testing.T) {
		t.Parallel()

		middleware := NewMiddleware(nil)

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			return []*rivertype.JobInsertResult{
				{Job: &rivertype.JobRow{ID: 123}},
			}, nil
		}

		_, err := middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		require.NoError(t, err)
	})

	t.Run("WorkSuccess", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return nil
		}

		err := middleware.Work(ctx, &rivertype.JobRow{Kind: "no_op"}, doInner)
		require.NoError(t, err)

		spans := bundle.exporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "river.work", span.Name)
		require.Equal(t, codes.Ok, span.Status.Code)
	})

	t.Run("WorkError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return errors.New("error from doInner")
		}

		err := middleware.Work(ctx, &rivertype.JobRow{Kind: "no_op"}, doInner)
		require.EqualError(t, err, "error from doInner")

		spans := bundle.exporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "river.work", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "error from doInner", span.Status.Description)
	})

	// Make sure the middleware can fall back to a global provider.
	t.Run("WorkEmptyConfig", func(t *testing.T) {
		t.Parallel()

		middleware := NewMiddleware(nil)

		doInner := func(ctx context.Context) error {
			return nil
		}

		err := middleware.Work(ctx, &rivertype.JobRow{Kind: "no_op"}, doInner)
		require.NoError(t, err)
	})
}
