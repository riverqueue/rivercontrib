package riverdatadog2

import (
	"context"
	"testing"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/ext"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/stretchr/testify/require"

	"github.com/riverqueue/river/rivertype"
)

// Verify interface compliance.
var (
	_ rivertype.JobInsertMiddleware = &Middleware{}
	_ rivertype.WorkerMiddleware    = &Middleware{}
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type testBundle struct{}

	setup := func(t *testing.T, config *MiddlewareConfig) (*Middleware, *testBundle) {
		t.Helper()

		require.NoError(t, tracer.Start())
		t.Cleanup(tracer.Stop)

		return NewMiddleware(config), &testBundle{}
	}

	t.Run("InsertMany", func(t *testing.T) {
		t.Parallel()

		middleware, _ := setup(t, nil)

		var span *tracer.Span

		doInner := func(ctx context.Context) ([]*rivertype.JobInsertResult, error) {
			span, _ = tracer.SpanFromContext(ctx)

			return []*rivertype.JobInsertResult{
				{Job: &rivertype.JobRow{ID: 123}},
			}, nil
		}

		insertRes, err := middleware.InsertMany(ctx, []*rivertype.JobInsertParams{{Kind: "no_op"}}, doInner)
		require.NoError(t, err)
		require.Equal(t, []*rivertype.JobInsertResult{
			{Job: &rivertype.JobRow{ID: 123}},
		}, insertRes)

		require.NotNil(t, span)
		spanMap := span.AsMap() // DataDog makes getting information out in any other way difficult/impossible
		require.Equal(t, "river.insert_many", spanMap[ext.SpanName])
	})
}
