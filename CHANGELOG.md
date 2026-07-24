# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add `otelriver` hook support for River producer metrics, including `river.job_get_available_duration` and `river.job_get_available_count`. [PR #64](https://github.com/riverqueue/rivercontrib/pull/64).

## [0.11.0] - 2026-07-02

### Added

- Add `EnableTracePropagation` configuration method, which enables linking river job spans to the span that enqueued them for tracing *why* a job was enqueued. [PR #65](https://github.com/riverqueue/rivercontrib/pull/65)

## [0.10.0] - 2026-06-06

### Added

- Add `kinds` span attribute to `otelriver` `insert_many` spans listing the distinct job kinds in each batch. [PR #62](https://github.com/riverqueue/rivercontrib/pull/62).

## [0.9.0] - 2026-06-02

### Added

- Added `unique_skipped_as_duplicate` attributes to otel `insert_many` spans and `insert_count` metric. [PR #58](https://github.com/riverqueue/rivercontrib/pull/58).

### Changed

- Record snoozed jobs with status `ok` instead of `error` in `otelriver` middleware. Add new `snooze.duration` span attribute. [PR #59](https://github.com/riverqueue/rivercontrib/pull/59).

## [0.8.0] - 2026-05-15

### Added

- Augment `otelriver` middleware to cleanly handle errors returned in batch results from River Pro's batch jobs feature. [PR #54](https://github.com/riverqueue/rivercontrib/pull/54).

## [0.7.0] - 2026-01-18

### Added

- Add `versionedjob` hook providing a simple declarative version change framework for jobs. [PR #39](https://github.com/riverqueue/rivercontrib/pull/39).

## [0.6.0] - 2025-11-20

### Added

- Add `panictoerror` middleware that recovers panics and returns them as errors to middlewares up the stack. [PR #32](https://github.com/riverqueue/rivercontrib/pull/32).

### Changed

- More complete example test for `nilerror` package. [PR #27](https://github.com/riverqueue/rivercontrib/pull/27).

### Fixed

- Fix GitHub import path for `nilerror` package. [PR #29](https://github.com/riverqueue/rivercontrib/pull/29).
- Fix compliance of `nilerror.Hook` to `rivertype.HookWorkEnd` interface. [PR #30](https://github.com/riverqueue/rivercontrib/pull/30).

## [0.5.0] - 2025-05-02

### Added

- Add `nilerror` hook to detect nil structs wrapped in non-nil error interfaces. [PR #25](https://github.com/riverqueue/rivercontrib/pull/25).

## [0.4.0] - 2025-04-18

### Added

- Added `otelriver` option `EnableWorkSpanJobKindSuffix` which appends the job kind a suffix to work spans so they look like `river.work/my_job` instead of `river.work`. [PR #23](https://github.com/riverqueue/rivercontrib/pull/23).

## [0.3.0] - 2025-04-14

### Added

- Added license file (`LICENSE`). The project uses the same Mozilla Public License Version 2.0 that the main River project uses. [PR #19](https://github.com/riverqueue/rivercontrib/pull/19).

## [0.2.0] - 2025-04-06

### Added

- Added `otelriver` option `MiddlewareConfig.DurationUnit`. Can be used to configure duration metrics to be emitted in milliseconds instead of the default seconds. [PR #10](https://github.com/riverqueue/rivercontrib/pull/10).
- More attributes like job ID and timestamps on OpenTelemetry spans. [PR #11](https://github.com/riverqueue/rivercontrib/pull/11).
- Added `otelriver` option `EnableSemanticMetrics` which will cause the middleware to emit metrics compliant with OpenTelemetry [semantic conventions](https://opentelemetry.io/docs/specs/semconv/messaging/messaging-metrics/). [PR #12](https://github.com/riverqueue/rivercontrib/pull/12).

## [0.1.0] - 2025-03-16

### Added

- Initial release. Mainly brings in the `otelriver` package for use of River with OpenTelemetry and DataDog. [PR #1](https://github.com/riverqueue/rivercontrib/pull/1).
