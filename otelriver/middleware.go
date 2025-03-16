// Package otelriver provides OpenTelemetry utilities for the River job
// queue.
package otelriver

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	// OpenTelemetry docs recommended this be a fully qualified Go package name.
	name = "github.com/riverqueue/rivercontrib/otelriver"

	prefix = "river."
)

// MiddlewareConfig is configuration for River's OpenTelemetry middleware.
type MiddlewareConfig struct {
	// MeterProvider is a MetricProvider to base metrics on. May be left as nil
	// to use the default global provider.
	MeterProvider metric.MeterProvider

	// TracerProvider is a TracerProvider to base traces on. May be left as nil
	// to use the default global provider.
	TracerProvider trace.TracerProvider
}

// Middleware is a River middleware that emits OpenTelemetry metrics when jobs
// are inserted or worked.
type Middleware struct {
	river.MiddlewareDefaults

	meter   metric.Meter
	metrics middlewareMetrics
	tracer  trace.Tracer
}

// Bundle of metrics associated with a middleware.
type middlewareMetrics struct {
	jobWork                    metric.Int64Counter
	jobWorkDuration            metric.Float64Gauge
	jobWorkDurationHistogram   metric.Float64Histogram
	jobInsert                  metric.Int64Counter
	jobInsertDuration          metric.Float64Gauge
	jobInsertDurationHistogram metric.Float64Histogram
}

// NewMiddleware initializes a new River OpenTelemetry middleware.
//
// config may be nil.
func NewMiddleware(config *MiddlewareConfig) *Middleware {
	var (
		meterProvider  = otel.GetMeterProvider()
		tracerProvider = otel.GetTracerProvider()
	)
	if config != nil {
		if config.MeterProvider != nil {
			meterProvider = config.MeterProvider
		}
		if config.TracerProvider != nil {
			tracerProvider = config.TracerProvider
		}
	}

	meter := meterProvider.Meter(name)

	return &Middleware{
		meter: meter,
		metrics: middlewareMetrics{
			// See unit guidelines:
			//
			// https://opentelemetry.io/docs/specs/semconv/general/metrics/#instrument-units
			jobInsert:                  mustInt64Counter(meter, prefix+"job_insert", metric.WithDescription("Number of jobs inserted"), metric.WithUnit("{job}")),
			jobInsertDuration:          mustFloat64Gauge(meter, prefix+"job_insert_duration", metric.WithDescription("Duration of insertion of job batch"), metric.WithUnit("s")),
			jobInsertDurationHistogram: mustFloat64Histogram(meter, prefix+"job_insert_duration_histogram", metric.WithDescription("Duration of insertion of job batch (histogram)"), metric.WithUnit("s")),
			jobWork:                    mustInt64Counter(meter, prefix+"job_work", metric.WithDescription("Number of jobs worked"), metric.WithUnit("{job}")),
			jobWorkDuration:            mustFloat64Gauge(meter, prefix+"job_work_duration", metric.WithDescription("Duration of job being worked"), metric.WithUnit("s")),
			jobWorkDurationHistogram:   mustFloat64Histogram(meter, prefix+"job_work_duration_histogram", metric.WithDescription("Duration of job being worked (histogram)"), metric.WithUnit("s")),
		},
		tracer: tracerProvider.Tracer(name),
	}
}

func (m *Middleware) InsertMany(ctx context.Context, manyParams []*rivertype.JobInsertParams, doInner func(ctx context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	ctx, span := m.tracer.Start(ctx, prefix+"insert_many")
	defer span.End()

	attrs := []attribute.KeyValue{
		attribute.String("status", ""), // replaced below
	}
	const statusIndex = 0

	var (
		begin     = time.Now()
		err       error
		insertRes []*rivertype.JobInsertResult
		panicked  = true // set to false if program leaves normally
	)
	defer func() {
		durationSeconds := time.Since(begin).Seconds()

		setAttributeAndSpanStatus(attrs, statusIndex, span, panicked, err)

		// This allocates a new slice, so make sure to do it as few times as possible.
		measurementOpt := metric.WithAttributes(attrs...)

		m.metrics.jobInsert.Add(ctx, int64(len(manyParams)))
		m.metrics.jobInsertDuration.Record(ctx, durationSeconds, measurementOpt)
		m.metrics.jobInsertDurationHistogram.Record(ctx, durationSeconds, measurementOpt)
	}()

	insertRes, err = doInner(ctx)
	panicked = false
	return insertRes, err
}

func (m *Middleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	ctx, span := m.tracer.Start(ctx, prefix+"work")
	defer span.End()

	attrs := []attribute.KeyValue{
		attribute.Int("attempt", job.Attempt),
		attribute.String("kind", job.Kind),
		attribute.String("queue", job.Queue),
		attribute.String("status", ""), // replaced below
	}
	const statusIndex = 3

	var (
		begin    = time.Now()
		err      error
		panicked = true // set to false if program leaves normally
	)
	defer func() {
		durationSeconds := time.Since(begin).Seconds()

		setAttributeAndSpanStatus(attrs, statusIndex, span, panicked, err)

		// This allocates a new slice, so make sure to do it as few times as possible.
		measurementOpt := metric.WithAttributes(attrs...)

		m.metrics.jobWork.Add(ctx, 1, measurementOpt)
		m.metrics.jobWorkDuration.Record(ctx, durationSeconds, measurementOpt)
		m.metrics.jobWorkDurationHistogram.Record(ctx, durationSeconds, measurementOpt)
	}()

	err = doInner(ctx)
	panicked = false
	return err
}

func mustFloat64Gauge(meter metric.Meter, name string, options ...metric.Float64GaugeOption) metric.Float64Gauge {
	metric, err := meter.Float64Gauge(name, options...)
	if err != nil {
		panic(err)
	}
	return metric
}

func mustFloat64Histogram(meter metric.Meter, name string, options ...metric.Float64HistogramOption) metric.Float64Histogram {
	metric, err := meter.Float64Histogram(name, options...)
	if err != nil {
		panic(err)
	}
	return metric
}

func mustInt64Counter(meter metric.Meter, name string, options ...metric.Int64CounterOption) metric.Int64Counter {
	metric, err := meter.Int64Counter(name, options...)
	if err != nil {
		panic(err)
	}
	return metric
}

// Sets success status on the given span and within the set of attributes. The
// index of the status attribute is required ahead of time as a minor
// optimization.
func setAttributeAndSpanStatus(attrs []attribute.KeyValue, statusIndex int, span trace.Span, panicked bool, err error) {
	if attrs[statusIndex].Key != "status" {
		panic("status attribute not at expected index; bug?") // protect against future regression
	}

	switch {
	case panicked:
		attrs[statusIndex] = attribute.String("status", "panic")
		span.SetStatus(codes.Error, "panic")
	case err != nil:
		attrs[statusIndex] = attribute.String("status", "error")
		span.SetStatus(codes.Error, err.Error())
	default:
		attrs[statusIndex] = attribute.String("status", "ok")
		span.SetStatus(codes.Ok, "")
	}
	span.SetAttributes(attrs...) // set after finalizing status
}
