package monitoring

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := new("testNew")

	require.NotNil(t, m.Counter)
	require.NotNil(t, m.Duration)
}

func TestNewAppMetrics(t *testing.T) {
	m := NewAppMetrics()

	require.NotNil(t, m)

	require.NotNil(t, m.RedisMetrics)
	require.NotNil(t, m.PostgresMetrics)
	require.NotNil(t, m.WorkerMetrics)
	require.NotNil(t, m.SMTPMetrics)
	require.NotNil(t, m.ListNotificationMetrics)
	require.NotNil(t, m.SendNotificationMetrics)
	require.NotNil(t, m.SendNotificationViaTimeMetrics)
}

func TestInc(t *testing.T) {
	m := new("testInc")

	tests := []struct {
		name    string
		typeInc func()
		status  string
	}{
		{
			name: "success",
			typeInc: func() {
				m.IncSuccess("operation")
			},
			status: StatusSuccess,
		},
		{
			name: "error",
			typeInc: func() {
				m.IncError("operation")
			},
			status: StatusError,
		},
		{
			name: "canceled",
			typeInc: func() {
				m.IncCanceled("operation")
			},
			status: StatusCanceled,
		},
		{
			name: "timeout",
			typeInc: func() {
				m.IncTimeout("operation")
			},
			status: StatusTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.typeInc()

			count := testutil.ToFloat64(m.Counter.WithLabelValues("operation", tt.status))
			require.Equal(t, float64(1), count)

			tt.typeInc()
			tt.typeInc()

			count = testutil.ToFloat64(m.Counter.WithLabelValues("operation", tt.status))
			require.Equal(t, float64(3), count)
		})
	}
}

func TestObserve(t *testing.T) {
	m := new("testObserve")

	m.Observe("operation", time.Now())

	count := testutil.CollectAndCount(m.Duration)
	require.NotEqual(t, 0, count)
}

func TestNop(t *testing.T) {
	m := NewNop()

	m.IncSuccess("operation")
	m.IncError("operation")
	m.IncCanceled("operation")
	m.IncTimeout("operation")
	m.Observe("operation", time.Now())
}
