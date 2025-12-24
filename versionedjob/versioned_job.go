package versionedjob

import (
	"context"

	"github.com/riverqueue/river/rivershared/baseservice"
	"github.com/riverqueue/river/rivertype"
)

// VersionTransformer defines how to perform transformations between versions
// for a specific job kind.
type VersionTransformer interface {
	// Kind is the job kind that this transformer applies to.
	Kind() string

	// VersionTransform applies version transformations to the given job.
	// Version transformations are defined according to the user, as well as how
	// a version is extracted from the job's args.
	//
	// Generally, this function should extract a version from the job, then
	// apply versions one by one until it's fully modernized to the point where
	// it can be successfully run by its worker.
	VersionTransform(ctx context.Context, job *rivertype.JobRow) error
}

// Verify interface compliance.
var _ rivertype.HookWorkBegin = &Hook{}

// HookConfig is configuration for the versionedjob hook.
type HookConfig struct {
	// Transformers are version transformers that the hook will apply. Only one
	// version transformer should be registered for any particular job kind.
	Transformers []VersionTransformer
}

// Hook is a River hook that applies version transformations on jobs so that
// workers can be written to handle only the most modern version, keeping worker
// code simple and clean.
type Hook struct {
	baseservice.BaseService
	rivertype.Hook

	config          *HookConfig
	transformersMap map[string]VersionTransformer
}

// NewHook initializes a new River versionedjob hook.
//
// config may be nil.
func NewHook(config *HookConfig) *Hook {
	if config == nil {
		config = &HookConfig{}
	}

	transformersMap := make(map[string]VersionTransformer, len(config.Transformers))
	for _, transformer := range config.Transformers {
		if _, ok := transformersMap[transformer.Kind()]; ok {
			panic("duplicate version transformer for kind: " + transformer.Kind())
		}

		transformersMap[transformer.Kind()] = transformer
	}

	return &Hook{
		config:          config,
		transformersMap: transformersMap,
	}
}

func (h *Hook) WorkBegin(ctx context.Context, job *rivertype.JobRow) error {
	if transformer, ok := h.transformersMap[job.Kind]; ok {
		return transformer.VersionTransform(ctx, job)
	}

	return nil
}
