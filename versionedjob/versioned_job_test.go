package versionedjob_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivershared/riversharedtest"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/versionedjob"
)

func TestHook(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	type testBundle struct{}

	setupConfig := func(t *testing.T, config *versionedjob.HookConfig) (*versionedjob.Hook, *testBundle) {
		t.Helper()

		return baseservice.Init(
			riversharedtest.BaseServiceArchetype(t),
			versionedjob.NewHook(config),
		), &testBundle{}
	}

	setup := func(t *testing.T) (*versionedjob.Hook, *testBundle) {
		t.Helper()

		return setupConfig(t, &versionedjob.HookConfig{
			Transformers: []versionedjob.VersionTransformer{
				&VersionedJobTransformer{},
			},
		})
	}

	t.Run("CurrentVersionNoOp", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		job := &rivertype.JobRow{
			EncodedArgs: mustMarshalJSON(t, map[string]any{
				"title":       "My Job",
				"description": "A description of a My Job.",
				"version":     3,
			}),
			Kind: (VersionedJobArgs{}).Kind(),
		}

		require.NoError(t, hook.WorkBegin(ctx, job))

		// Expect no changes.
		require.Equal(t, VersionedJobArgs{
			Title:       "My Job",
			Description: "A description of a My Job.",
			Version:     3,
		}, mustUnmarshalJSON[VersionedJobArgs](t, job.EncodedArgs))
	})

	t.Run("AppliesVersion", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		job := &rivertype.JobRow{
			EncodedArgs: mustMarshalJSON(t, map[string]any{
				"title":   "My Job",
				"version": 2,
			}),
			Kind: (VersionedJobArgs{}).Kind(),
		}

		require.NoError(t, hook.WorkBegin(ctx, job))

		// Expect latest version.
		require.Equal(t, VersionedJobArgs{
			Title:       "My Job",
			Description: "A description of a My Job.",
			Version:     3,
		}, mustUnmarshalJSON[VersionedJobArgs](t, job.EncodedArgs))
	})

	t.Run("AppliesMultipleVersions", func(t *testing.T) {
		t.Parallel()

		hook, _ := setup(t)

		job := &rivertype.JobRow{
			EncodedArgs: mustMarshalJSON(t, map[string]any{
				"name": "My Job",
				// notably, version is absent here because job rows likely start life without one
			}),
			Kind: (VersionedJobArgs{}).Kind(),
		}

		require.NoError(t, hook.WorkBegin(ctx, job))

		// Expect latest version.
		require.Equal(t, VersionedJobArgs{
			Title:       "My Job",
			Description: "A description of a My Job.",
			Version:     3,
		}, mustUnmarshalJSON[VersionedJobArgs](t, job.EncodedArgs))
	})
}

func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err)

	return data
}

func mustUnmarshalJSON[T any](t *testing.T, data []byte) T {
	t.Helper()

	var v T
	require.NoError(t, json.Unmarshal(data, &v))

	return v
}
