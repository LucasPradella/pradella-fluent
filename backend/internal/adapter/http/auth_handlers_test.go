package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterContract(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	resp := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "ana@example.com", "password": "supersecret123", "displayName": "Ana",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body := decode(t, resp)
	assert.Equal(t, "ana@example.com", body["email"])
	assert.Equal(t, "Ana", body["displayName"])
	assert.Nil(t, body["proficiencyLevel"], "level is null before placement")
	assert.NotEmpty(t, body["id"])

	// Session cookie HttpOnly; CSRF cookie readable.
	var session, csrf *http.Cookie
	for _, ck := range resp.Cookies() {
		switch ck.Name {
		case "fluentdev_session":
			session = ck
		case "fluentdev_csrf":
			csrf = ck
		}
	}
	require.NotNil(t, session)
	assert.True(t, session.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, session.SameSite)
	require.NotNil(t, csrf)
	assert.False(t, csrf.HttpOnly)
}

func TestRegisterDuplicate409(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	resp := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "ana@example.com", "password": "supersecret123", "displayName": "Ana 2",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
	body := decode(t, resp)
	assert.EqualValues(t, 409, body["status"])
}

func TestLoginContract(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	// Wrong password → 401 problem+json, generic title.
	resp := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "ana@example.com", "password": "wrong-password",
	})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	body := decode(t, resp)
	resp.Body.Close()
	assert.NotContains(t, strings.ToLower(body["detail"].(string)), "não existe",
		"error must not disclose whether the account exists")

	// Correct password → 200 with user body.
	resp = c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "ana@example.com", "password": "supersecret123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ana@example.com", decode(t, resp)["email"])
}

func TestMeRequiresSession(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	resp := c.do(http.MethodGet, "/api/v1/me", nil)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	c.register("ana@example.com", "Ana")
	resp = c.do(http.MethodGet, "/api/v1/me", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Ana", decode(t, resp)["displayName"])
}

func TestLogoutRevokesSession(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	resp := c.do(http.MethodPost, "/api/v1/auth/logout", nil)
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp = c.do(http.MethodGet, "/api/v1/me", nil)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCSRFRequiredOnStateChanges(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	// Hand-build a POST without the X-CSRF-Token header.
	req, err := http.NewRequest(http.MethodPost, c.base+"/api/v1/placement/session", nil)
	require.NoError(t, err)
	resp, err := c.http.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "missing CSRF token must be rejected")
}

func TestAuthRateLimit429(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	got429 := false
	for i := 0; i < 15; i++ {
		resp := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
			"email": "burst@example.com", "password": "wrong-password-x",
		})
		if resp.StatusCode == http.StatusTooManyRequests {
			assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
			got429 = true
		}
		resp.Body.Close()
	}
	assert.True(t, got429, "login burst must trigger 429 (OWASP A07)")
}

func TestSecurityHeadersPresent(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)

	resp := c.do(http.MethodGet, "/api/v1/me", nil)
	defer resp.Body.Close()
	assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.NotEmpty(t, resp.Header.Get("Permissions-Policy"))
	assert.NotEmpty(t, resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestUnknownRoute404ProblemJSON(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	resp := c.do(http.MethodGet, "/api/v1/nope", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}
