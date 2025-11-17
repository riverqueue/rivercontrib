package panictoerror

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/riverqueue/river/rivertype"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	type testBundle struct{}

	setupConfig := func(t *testing.T, config *MiddlewareConfig) (*Middleware, *testBundle) {
		t.Helper()

		return baseservice.Init(
			riversharedtest.BaseServiceArchetype(t),
			NewMiddleware(config),
		), &testBundle{}
	}

	setup := func(t *testing.T) (*Middleware, *testBundle) {
		t.Helper()

		return setupConfig(t, nil)
	}

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()

		middleware, _ := setup(t)

		require.NoError(t, middleware.Work(ctx, &rivertype.JobRow{}, func(context.Context) error { return nil }))
	})

	t.Run("InnerMiddlewareReturnsError", func(t *testing.T) {
		t.Parallel()

		middleware, _ := setup(t)

		expectedErr := errors.New("my error")

		require.ErrorIs(t, middleware.Work(ctx, &rivertype.JobRow{}, func(context.Context) error {
			return expectedErr
		}), expectedErr)
	})

	t.Run("PanicReturnedAsError", func(t *testing.T) {
		t.Parallel()

		middleware, _ := setup(t)

		err := middleware.Work(ctx, &rivertype.JobRow{}, func(context.Context) error {
			panic("my panic")
		})

		var panicErr *PanicError
		require.ErrorAs(t, err, &panicErr)
		require.Equal(t, "my panic", panicErr.Cause)

		t.Log(panicErr.Error())

		// Looking for this function to be the top of trace (i.e. we skipped all
		// the internal frames that were in there).
		require.Contains(t, panicErr.Trace[0].Function, "TestMiddleware")
	})
}

func TestPanicErrorIs(t *testing.T) {
	t.Parallel()

	err := &PanicError{}
	require.ErrorIs(t, err, &PanicError{})
}
