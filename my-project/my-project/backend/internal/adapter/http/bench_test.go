package httpapi_test

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// TestNonSpeechLatencyBudget exercises the full middleware + handler chain
// in-process and asserts p95 < 200 ms for non-speech endpoints (plan
// performance goal). Repository latency is out of scope here (covered by
// indexes + integration tests); this guards against regressions in the
// request pipeline itself.
func TestNonSpeechLatencyBudget(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "perf@example.com", domain.LevelBasic)

	paths := []string{"/api/v1/me", "/api/v1/tracks", "/api/v1/dashboard"}
	const rounds = 50

	var samples []time.Duration
	for i := 0; i < rounds; i++ {
		for _, p := range paths {
			start := time.Now()
			resp := c.do(http.MethodGet, p, nil)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
			samples = append(samples, time.Since(start))
		}
	}

	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	p95 := samples[int(float64(len(samples))*0.95)-1]
	t.Logf("p95 over %d requests: %s", len(samples), p95)
	require.Less(t, p95, 200*time.Millisecond, "non-speech endpoints must stay under the 200 ms p95 budget")
}
