package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Observability checks (T073 / OWASP A09): structured logs carry request
// ids but never PII, credentials or audio.

func TestLogsCarryRequestIDButNoPII(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	resp := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "sensitive-pii@example.com", "password": "ultra-secret-password-1", "displayName": "Ana",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	requestID := resp.Header.Get("X-Request-ID")
	resp.Body.Close()

	logs := h.logBuf.String()
	assert.Contains(t, logs, `"request_id":"`+requestID+`"`, "request id must be logged for correlation")
	assert.Contains(t, logs, `"path":"/api/v1/auth/register"`)
	assert.NotContains(t, logs, "sensitive-pii@example.com", "e-mails must never be logged")
	assert.NotContains(t, logs, "ultra-secret-password-1", "credentials must never be logged")
}

func TestRateLimitEventsAreLogged(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	for i := 0; i < 15; i++ {
		resp := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
			"email": "burst@example.com", "password": "wrong-password-x",
		})
		resp.Body.Close()
	}
	assert.Contains(t, h.logBuf.String(), "rate limit exceeded",
		"rate-limit rejections must emit an auditable event")
}
