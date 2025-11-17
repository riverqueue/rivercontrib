# nilerror [![Build Status](https://github.com/riverqueue/rivercontrib/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/riverqueue/rivercontrib/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/riverqueue/rivercontrib.svg)](https://pkg.go.dev/github.com/riverqueue/rivercontrib/nilerror)

Provides a River hook for detecting a common accidental Go problem where a nil struct value is wrapped in a non-nil interface value. This commonly causes trouble with the error interface, where an unintentional non-nil error is returned. For example:

``` go
func returnsError() error {
    var p *MyError = nil
    if bad() {
        p = ErrBad
    }
    return p // Will always return a non-nil error.
}
```

See https://go.dev/doc/faq#nil_error.

See [`example_hook_test.go`](./example_hook_test.go) for usage details.

## Options

The hook supports these options:

``` go
hook := nilerror.NewHook(&HookConfig{
    Suppress: true,
})
```

* `Suppress`: Causes the hook to suppress detected nil struct values wrapped in non-nil error interface values and produce warning logging instead.