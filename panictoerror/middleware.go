// Package panictoerror provides a rivertype.WorkerMiddleware that recovers
// panics that may have occurred deeper in the middleware stack (i.e. an inner
// middleware or the worker itself), converts those panics to errors, and
// returns those errors up the stack. This may be convenient in some cases so
// that middleware further up the stack need only have one way to handle either
// return errors or panic values.
package panictoerror

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivertype"
)

// Verify interface compliance.
var _ rivertype.WorkerMiddleware = &Middleware{}

// PanicError is a panic that's been converted to an error.
type PanicError struct {
	// Cause is the value recovered with `recover()`.
	Cause any

	// Trace up to the top 100 stack frames when the panic occurred. The
	// middleware attempts to remove internal frames on top so that user code is
	// the first stack frame.
	Trace []*runtime.Frame
}

func (e *PanicError) Error() string {
	var sb strings.Builder
	for _, frame := range e.Trace {
		sb.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
	}

	return fmt.Sprintf("PanicError: %v\n%s", e.Cause, sb.String())
}

func (e *PanicError) Is(target error) bool {
	_, ok := target.(*PanicError)
	return ok
}

// MiddlewareConfig is configuration for the panictoerror middleware.
//
// Currently empty, but reserved for future use.
type MiddlewareConfig struct{}

// Middleware is a rivertype.WorkerMiddleware that recovers panics that may have
// occurred deeper in the middleware stack (i.e. an inner middleware or the
// worker itself), converts those panics to errors, and returns those errors up
// the stack.
type Middleware struct {
	baseservice.BaseService
	river.MiddlewareDefaults

	config *MiddlewareConfig
}

// NewMiddleware initializes a new River panictoerror middleware.
//
// config may be nil.
func NewMiddleware(config *MiddlewareConfig) *Middleware {
	if config == nil {
		config = &MiddlewareConfig{}
	}

	return &Middleware{
		config: config,
	}
}

func (s *Middleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) (err error) {
	defer func() {
		if recovery := recover(); recovery != nil {
			err = &PanicError{
				Cause: recovery,

				// Skip (1) Callers, (2) captureStackTraceSkipFrames, (3) Work (this function), and (4) panic.go.
				//
				// runtime.Callers
				//     /opt/homebrew/Cellar/go/1.25.0/libexec/src/runtime/extern.go:345
				// github.com/riverqueue/rivercontrib/panictoerror.captureStackTraceSkipFrames
				//     /Users/brandur/Documents/projects/rivercontrib/panictoerror/middleware.go:77
				// github.com/riverqueue/rivercontrib/panictoerror.(*Middleware).Work.func1
				//     /Users/brandur/Documents/projects/rivercontrib/panictoerror/middleware.go:58
				// runtime.gopanic
				//     /opt/homebrew/Cellar/go/1.25.0/libexec/src/runtime/panic.go:783
				Trace: captureStackFrames(4),
			}
		}
	}()

	err = doInner(ctx)
	return err
}

// captureStackFrames captures the current stack trace, skipping the top
// numSkipped frames.
func captureStackFrames(numSkipped int) []*runtime.Frame {
	var (
		// Allocate room for up to 100 callers; adjust as needed.
		callers = make([]uintptr, 100)

		// Skip the specified number of frames.
		numFrames = runtime.Callers(numSkipped, callers)

		frames = runtime.CallersFrames(callers[:numFrames])
	)

	trace := make([]*runtime.Frame, 0, numFrames)
	for {
		frame, more := frames.Next()
		trace = append(trace, &frame)
		if !more {
			break
		}
	}
	return trace
}
