# otelriver

[OpenTelemetry](https://opentelemetry.io/) utilities for the [River job queue](https://github.com/riverqueue/river).

See [`example_middleware_test.go`](./example_middleware_test.go) for usage details.

## Options

The middleware supports these options:

``` go
middleware := otelriver.NewMiddleware(&MiddlewareConfig{
    DurationUnit:          "ms",
    EnableSemanticMetrics: true,
    MeterProvider:         meterProvider,
    TracerProvider:        tracerProvider,
})
```

* `DurationUnit`: The unit which durations are emitted as, either "ms" (milliseconds) or "s" (seconds). Defaults to seconds.
* `EnableSemanticMetrics`: Causes the middleware to emit metrics compliant with OpenTelemetry's ["semantic conventions"](https://opentelemetry.io/docs/specs/semconv/messaging/messaging-metrics/) for message clients. This has the effect of having all messaging systems share the same common metric names, with attributes differentiating them.
* `MeterProvider`: Injected OpenTelemetry meter provider. The global meter provider is used by default.
* `TracerProvider`: Injected OpenTelemetry tracer provider. The global tracer provider is used by default.

## Use with DataDog

See [using the OpenTelemetry API with DataDog](https://docs.datadoghq.com/tracing/trace_collection/custom_instrumentation/go/otel/) and the examples in [`datadogriver`](../datadogriver/) for how to configure a DataDog OpenTelemetry tracer provider.
