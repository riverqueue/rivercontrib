package otelriver

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
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

	ctx := context.Background()

	type testBundle struct {
		metricReader  *metric.ManualReader
		traceExporter *tracetest.InMemoryExporter
	}

	setupConfig := func(t *testing.T, config *MiddlewareConfig) (*Middleware, *testBundle) {
		t.Helper()

		var (
			metricReader  = metric.NewManualReader()
			traceExporter = tracetest.NewInMemoryExporter()
		)

		config.MeterProvider = metric.NewMeterProvider(metric.WithReader(metricReader))
		config.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSyncer(traceExporter))

		return NewMiddleware(config), &testBundle{
			metricReader:  metricReader,
			traceExporter: traceExporter,
		}
	}

	setup := func(t *testing.T) (*Middleware, *testBundle) {
		t.Helper()

		return setupConfig(t, &MiddlewareConfig{})
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

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "ok", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, "river.insert_many", span.Name)
		require.Equal(t, codes.Ok, span.Status.Code)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "ok"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.insert_count", 1, expectedAttrs...)
		requireSum(t, metrics, "river.insert_many_count", 1, expectedAttrs...)
		{
			metric, _ := requireGaugeNotEmpty(t, metrics, "river.insert_many_duration", expectedAttrs...)
			require.Equal(t, "s", metric.Unit)
		}
		{
			metric, _ := requireHistogramCount(t, metrics, "river.insert_many_duration_histogram", 1, expectedAttrs...)
			require.Equal(t, "s", metric.Unit)
		}
	})

	t.Run("InsertManyError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			return nil, errors.New("error from doInner")
		}

		_, err := middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		require.EqualError(t, err, "error from doInner")

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "error", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, "river.insert_many", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "error from doInner", span.Status.Description)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "error"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.insert_count", 1, expectedAttrs...)
		requireSum(t, metrics, "river.insert_many_count", 1, expectedAttrs...)
		requireGaugeNotEmpty(t, metrics, "river.insert_many_duration", expectedAttrs...)
		requireHistogramCount(t, metrics, "river.insert_many_duration_histogram", 1, expectedAttrs...)
	})

	t.Run("InsertManyPanic", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			panic("panic from doInner")
		}

		require.PanicsWithValue(t, "panic from doInner", func() {
			_, _ = middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		})

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, "panic", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, "river.insert_many", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "panic", span.Status.Description)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "panic"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.insert_count", 1, expectedAttrs...)
		requireSum(t, metrics, "river.insert_many_count", 1, expectedAttrs...)
		requireGaugeNotEmpty(t, metrics, "river.insert_many_duration", expectedAttrs...)
		requireHistogramCount(t, metrics, "river.insert_many_duration_histogram", 1, expectedAttrs...)
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

	t.Run("InsertManyDurationUnitMS", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setupConfig(t, &MiddlewareConfig{
			DurationUnit: "ms",
		})

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

		var metrics metricdata.ResourceMetrics
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.insert_count", 1)
		requireSum(t, metrics, "river.insert_many_count", 1)
		{
			metric, _ := requireGaugeNotEmpty(t, metrics, "river.insert_many_duration")
			require.Equal(t, "ms", metric.Unit)
		}
		{
			metric, _ := requireHistogramCount(t, metrics, "river.insert_many_duration_histogram", 1)
			require.Equal(t, "ms", metric.Unit)
		}
	})

	t.Run("WorkSuccess", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return nil
		}

		var (
			createdAt   = time.Now()
			scheduledAt = time.Now().Add(1 * time.Second)
		)
		err := middleware.Work(ctx, &rivertype.JobRow{
			ID:          123,
			Attempt:     6,
			CreatedAt:   createdAt,
			Kind:        "no_op",
			Priority:    1,
			Queue:       "my_queue",
			ScheduledAt: scheduledAt,
			Tags:        []string{"a", "b"},
		}, doInner)
		require.NoError(t, err)

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, int64(123), getAttribute(t, span.Attributes, "id").AsInt64())
		require.Equal(t, int64(6), getAttribute(t, span.Attributes, "attempt").AsInt64())
		require.Equal(t, createdAt.Format(time.RFC3339), getAttribute(t, span.Attributes, "created_at").AsString())
		require.Equal(t, "my_queue", getAttribute(t, span.Attributes, "queue").AsString())
		require.Equal(t, int64(1), getAttribute(t, span.Attributes, "priority").AsInt64())
		require.Equal(t, "ok", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, scheduledAt.Format(time.RFC3339), getAttribute(t, span.Attributes, "scheduled_at").AsString())
		require.Equal(t, []string{"a", "b"}, getAttribute(t, span.Attributes, "tag").AsStringSlice())
		require.Equal(t, "river.work", span.Name)
		require.Equal(t, codes.Ok, span.Status.Code)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "ok"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.work_count", 1, expectedAttrs...)
		{
			metric, _ := requireGaugeNotEmpty(t, metrics, "river.work_duration", expectedAttrs...)
			require.Equal(t, "s", metric.Unit)
		}
		{
			metric, _ := requireHistogramCount(t, metrics, "river.work_duration_histogram", 1, expectedAttrs...)
			require.Equal(t, "s", metric.Unit)
		}
	})

	t.Run("WorkError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return errors.New("error from doInner")
		}

		err := middleware.Work(ctx, &rivertype.JobRow{
			Attempt: 6,
			Kind:    "no_op",
			Queue:   "my_queue",
		}, doInner)
		require.EqualError(t, err, "error from doInner")

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, int64(6), getAttribute(t, span.Attributes, "attempt").AsInt64())
		require.Equal(t, "my_queue", getAttribute(t, span.Attributes, "queue").AsString())
		require.Equal(t, "error", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, "river.work", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "error from doInner", span.Status.Description)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "error"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.work_count", 1, expectedAttrs...)
		requireGaugeNotEmpty(t, metrics, "river.work_duration", expectedAttrs...)
		requireHistogramCount(t, metrics, "river.work_duration_histogram", 1, expectedAttrs...)
	})

	t.Run("JobCancelError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return fmt.Errorf("wrapped job cancel: %w", rivertype.JobCancel(errors.New("inner error")))
		}

		err := middleware.Work(ctx, &rivertype.JobRow{
			Kind: "no_op",
		}, doInner)
		require.EqualError(t, err, "wrapped job cancel: JobCancelError: inner error")

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.True(t, getAttribute(t, span.Attributes, "cancel").AsBool())
	})

	t.Run("JobSnoozeError", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			return fmt.Errorf("wrapped job snooze: %w", &rivertype.JobSnoozeError{})
		}

		err := middleware.Work(ctx, &rivertype.JobRow{
			Kind: "no_op",
		}, doInner)
		require.EqualError(t, err, "wrapped job snooze: JobSnoozeError: 0s")

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.True(t, getAttribute(t, span.Attributes, "snooze").AsBool())
	})

	t.Run("WorkPanic", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setup(t)

		doInner := func(ctx context.Context) error {
			panic("panic from doInner")
		}

		require.PanicsWithValue(t, "panic from doInner", func() {
			_ = middleware.Work(ctx, &rivertype.JobRow{
				Attempt: 6,
				Kind:    "no_op",
				Queue:   "my_queue",
			}, doInner)
		})

		spans := bundle.traceExporter.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		require.Equal(t, int64(6), getAttribute(t, span.Attributes, "attempt").AsInt64())
		require.Equal(t, "my_queue", getAttribute(t, span.Attributes, "queue").AsString())
		require.Equal(t, "panic", getAttribute(t, span.Attributes, "status").AsString())
		require.Equal(t, "river.work", span.Name)
		require.Equal(t, codes.Error, span.Status.Code)
		require.Equal(t, "panic", span.Status.Description)

		var (
			expectedAttrs = []attribute.KeyValue{
				attribute.String("status", "panic"),
			}
			metrics metricdata.ResourceMetrics
		)
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.work_count", 1, expectedAttrs...)
		requireGaugeNotEmpty(t, metrics, "river.work_duration", expectedAttrs...)
		requireHistogramCount(t, metrics, "river.work_duration_histogram", 1, expectedAttrs...)
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

	t.Run("WorkDurationUnitMS", func(t *testing.T) {
		t.Parallel()

		middleware, bundle := setupConfig(t, &MiddlewareConfig{
			DurationUnit: "ms",
		})

		doInner := func(ctx context.Context) error {
			return nil
		}

		err := middleware.Work(ctx, &rivertype.JobRow{Kind: "no_op"}, doInner)
		require.NoError(t, err)

		var metrics metricdata.ResourceMetrics
		require.NoError(t, bundle.metricReader.Collect(ctx, &metrics))
		requireSum(t, metrics, "river.work_count", 1)
		{
			metric, _ := requireGaugeNotEmpty(t, metrics, "river.work_duration")
			require.Equal(t, "ms", metric.Unit)
		}
		{
			metric, _ := requireHistogramCount(t, metrics, "river.work_duration_histogram", 1)
			require.Equal(t, "ms", metric.Unit)
		}
	})
}

func getAttribute(t *testing.T, attrs []attribute.KeyValue, key string) attribute.Value {
	t.Helper()

	for _, attr := range attrs {
		if attr.Key == attribute.Key(key) {
			return attr.Value
		}
	}
	require.FailNow(t, "key not found in attributes: "+key)
	return attribute.Value{}
}

func getMetric[T metricdatatest.Datatypes](t *testing.T, metrics metricdata.ResourceMetrics, name string) (metricdata.Metrics, T) {
	t.Helper()

	for _, scopeMetrics := range metrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == name {
				return metric, metric.Data.(T) //nolint:forcetypeassert
			}
		}
	}
	t.Fatalf("Metrics not found: %s", name)
	var defaultVal T
	return metricdata.Metrics{}, defaultVal
}

func requireGaugeNotEmpty(t *testing.T, metrics metricdata.ResourceMetrics, name string, attrs ...attribute.KeyValue) (metricdata.Metrics, metricdata.Gauge[float64]) { //nolint:unparam
	t.Helper()

	metric, metricData := getMetric[metricdata.Gauge[float64]](t, metrics, name)
	require.NotEmpty(t, metricData.DataPoints)
	metricdatatest.AssertHasAttributes(t, metric, attrs...)
	return metric, metricData
}

func requireHistogramCount(t *testing.T, metrics metricdata.ResourceMetrics, name string, count uint64, attrs ...attribute.KeyValue) (metricdata.Metrics, metricdata.Histogram[float64]) { //nolint:unparam
	t.Helper()

	metric, metricData := getMetric[metricdata.Histogram[float64]](t, metrics, name)
	require.Equal(t, count, metricData.DataPoints[0].Count)
	metricdatatest.AssertHasAttributes(t, metric, attrs...)
	return metric, metricData
}

func requireSum(t *testing.T, metrics metricdata.ResourceMetrics, name string, val int64, attrs ...attribute.KeyValue) (metricdata.Metrics, metricdata.Sum[int64]) { //nolint:unparam
	t.Helper()

	metric, metricData := getMetric[metricdata.Sum[int64]](t, metrics, name)
	require.Equal(t, val, metricData.DataPoints[0].Value)
	metricdatatest.AssertHasAttributes(t, metric, attrs...)
	return metric, metricData
}
