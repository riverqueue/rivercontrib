# datadogriver

This package demonstrates the use of the [`otelriver`](../otelriver/) package with [DataDog's OpenTelemetry provider](https://docs.datadoghq.com/tracing/trace_collection/custom_instrumentation/go/otel/). This is provided as an alternative to direct integration with DataDog's Go SDK.

See:

* [`example_global_provider_test.go`](./example_global_provider_test.go): Usage with global tracer provider.
* [`example_injected_provider_test.go`](./example_injected_provider_test.go): Usage with tracer provider injected as configuration.
