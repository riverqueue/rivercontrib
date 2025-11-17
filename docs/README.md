# rivercontrib

[River](https://github.com/riverqueue/river) packages for third party systems.

See:

* [`datadogriver`](../datadogriver): Package containing examples of using `otelriver` with [DataDog](https://www.datadoghq.com/).
* [`nilerror`](../nilerror): Package containing a River hook for detecting a common accidental Go problem where a nil struct value is wrapped in a non-nil interface value.
* [`otelriver`](../otelriver): Package for use with [OpenTelemetry](https://opentelemetry.io/).
* [`panictoerror`](../panictoerror): Provides a middleware that recovers panics that may have occurred deeper in the middleware stack (i.e. an inner middleware or the worker itself), converts those panics to errors, and returns those errors up the stack.
