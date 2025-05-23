# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- More complete example test for `nilerror` package. [PR #27](https://github.com/riverqueue/rivercontrib/pull/27).

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
