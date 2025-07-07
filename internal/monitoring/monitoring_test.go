package monitoring

//import (
//	"testing"
//
//	"github.com/prometheus/client_golang/prometheus/testutil"
//	"github.com/stretchr/testify/require"
//)
//
//func TestNew(t *testing.T) {
//	m := new("testNew")
//
//	require.NotNil(t, m.Counter)
//	require.NotNil(t, m.Duration)
//}
//
//func TestInc(t *testing.T) {
//	m := new("testInc")
//
//	m.Inc("operation", StatusSuccess)
//
//	count := testutil.ToFloat64(m.Counter.WithLabelValues("operation", StatusSuccess))
//	require.Equal(t, float64(1), count)
//
//	m.Inc("operation", StatusSuccess)
//	m.Inc("operation", StatusSuccess)
//
//	count = testutil.ToFloat64(m.Counter.WithLabelValues("operation", StatusSuccess))
//	require.Equal(t, float64(3), count)
//}
//
//func TestObserve(t *testing.T) {
//	m := new("testObserve")
//
//	m.Observe("operation", 0.54)
//
//	count := testutil.CollectAndCount(m.Duration)
//	require.NotEqual(t, 0, count)
//}
//
//func TestNop(t *testing.T) {
//	m := NewNop()
//
//	m.Inc("operation", StatusSuccess)
//	m.Observe("operation", 0.54)
//}
