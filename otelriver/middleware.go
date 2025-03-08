// Package otelriver provides OpenTelemetry utilities for the River job
// queue.
package otelriver

import (
	"context"

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
	insertCount metric.Int64Counter
	workCount   metric.Int64Counter
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

	mustInt64Counter := func(name string, options ...metric.Int64CounterOption) metric.Int64Counter {
		metric, err := meter.Int64Counter(name, options...)
		if err != nil {
			panic(err)
		}
		return metric
	}

	return &Middleware{
		meter: meter,
		metrics: middlewareMetrics{
			insertCount: mustInt64Counter(prefix+"jobs_inserted", metric.WithDescription("Number of jobs inserted")),
			workCount:   mustInt64Counter(prefix+"jobs_worked", metric.WithDescription("Number of jobs worked")),
		},
		tracer: tracerProvider.Tracer(name),
	}
}

func (m *Middleware) InsertMany(ctx context.Context, manyParams []*rivertype.JobInsertParams, doInner func(ctx context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	ctx, span := m.tracer.Start(ctx, prefix+"insert_many")
	defer span.End()

	insertRes, err := doInner(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return insertRes, err
	}

	span.SetStatus(codes.Ok, "")
	m.metrics.insertCount.Add(ctx, int64(len(manyParams)))

	return insertRes, nil
}

func (m *Middleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	ctx, span := m.tracer.Start(ctx, prefix+"work")
	defer span.End()

	attributes := []attribute.KeyValue{
		attribute.Int("attempt", job.Attempt),
		attribute.String("kind", job.Kind),
		attribute.String("queue", job.Queue),
	}
	span.SetAttributes(attributes...)

	err := doInner(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	m.metrics.insertCount.Add(ctx, 1, metric.WithAttributes(attributes...))

	return err
}
