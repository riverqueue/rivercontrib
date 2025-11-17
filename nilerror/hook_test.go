package nilerror

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
)

type myCustomError struct{}

func (*myCustomError) Error() string {
	return "my custom error"
}

func TestHook(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	type testBundle struct{}

	setupConfig := func(t *testing.T, config *HookConfig) (*Hook, *testBundle) {
		t.Helper()

		return baseservice.Init(
			riversharedtest.BaseServiceArchetype(t),
			NewHook(config),
		), &testBundle{}
	}

	setup := func(t *testing.T) (*Hook, *testBundle) {
		t.Helper()

		return setupConfig(t, nil)
	}

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		require.NoError(t, hook.WorkEnd(ctx, &rivertype.JobRow{}, nil))
	})

	t.Run("NonNilError", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		myCustomErr := &myCustomError{}
		require.Equal(t, myCustomErr, hook.WorkEnd(ctx, &rivertype.JobRow{}, myCustomErr))
	})

	t.Run("NilError", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		var myCustomErr *myCustomError
		require.EqualError(t, hook.WorkEnd(ctx, &rivertype.JobRow{}, myCustomErr),
			"non-nil error containing nil internal value (see: https://go.dev/doc/faq#nil_error); probably a bug: (*nilerror.myCustomError)(<nil>)",
		)
	})

	t.Run("Suppress", func(t *testing.T) {
		t.Parallel()

		hook, _ := setupConfig(t, &HookConfig{Suppress: true})

		var logBuf bytes.Buffer
		hook.Logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn, ReplaceAttr: slogutil.NoLevelTime}))

		var myCustomErr *myCustomError
		require.NoError(t, hook.WorkEnd(ctx, &rivertype.JobRow{}, myCustomErr))

		require.Equal(t,
			`msg="nilerror.Hook: Got non-nil error containing nil internal value (see: https://go.dev/doc/faq#nil_error); probably a bug: (*nilerror.myCustomError)(<nil>)"`+"\n",
			logBuf.String())
	})
}
