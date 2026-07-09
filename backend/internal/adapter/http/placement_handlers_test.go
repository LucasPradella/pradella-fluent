package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlacementNoActiveSession404(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	resp := c.do(http.MethodGet, "/api/v1/placement/session", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestPlacementStartServesQuestionWithoutAnswerLeak(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	resp := c.do(http.MethodPost, "/api/v1/placement/session", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	assert.NotContains(t, string(raw), "correct", "correct answer must never reach the client")

	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	assert.Equal(t, "active", body["status"])
	assert.EqualValues(t, 0, body["questionsServed"])
	assert.Equal(t, "B1", body["currentBand"], "session starts at B1")
	q := body["nextQuestion"].(map[string]any)
	assert.NotEmpty(t, q["id"])
	assert.NotEmpty(t, q["options"])
}

// answerLoop submits answers until the session completes or n answers sent.
func answerLoop(t *testing.T, c *client, n int, answer func(i int) string) map[string]any {
	t.Helper()
	resp := c.do(http.MethodPost, "/api/v1/placement/session", nil)
	state := decode(t, resp)
	resp.Body.Close()

	for i := 0; i < n; i++ {
		if state["status"] == "completed" {
			break
		}
		q := state["nextQuestion"].(map[string]any)
		resp := c.do(http.MethodPost, "/api/v1/placement/session/answers", map[string]string{
			"questionId": q["id"].(string), "answer": answer(i),
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		state = decode(t, resp)
		resp.Body.Close()
	}
	return state
}

func TestPlacementFullWalkAllCorrectAssignsAdvanced(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	state := answerLoop(t, c, 12, func(int) string { return "ok" })

	assert.Equal(t, "completed", state["status"])
	assert.EqualValues(t, 12, state["questionsServed"], "hard stop at 12 questions")
	assert.Equal(t, "advanced", *jsonString(state, "assignedLevel"),
		"perfect walk B1→B2→C1→C1 must assign advanced")

	// users.proficiency_level assigned atomically.
	uid := h.userID(t, "ana@example.com")
	assert.EqualValues(t, "advanced", h.placement.LevelAssigned[uid])
}

func TestPlacementAllWrongAssignsBasic(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	state := answerLoop(t, c, 12, func(int) string { return "no" })
	assert.Equal(t, "completed", state["status"])
	assert.Equal(t, "basic", *jsonString(state, "assignedLevel"),
		"failing walk B1→A2→A1→A1 must assign basic")
}

func TestPlacementResumeKeepsProgress(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	answerLoop(t, c, 5, func(int) string { return "ok" })

	// Simulate app restart: new client, fresh login.
	c2 := h.client(t)
	resp := c2.do(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "ana@example.com", "password": "supersecret123",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = c2.do(http.MethodGet, "/api/v1/placement/session", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	state := decode(t, resp)
	assert.EqualValues(t, 5, state["questionsServed"], "resume at question 6 (US1 scenario 6)")
	assert.NotNil(t, state["nextQuestion"])
}

func TestPlacementRepeatAnswer409(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")

	resp := c.do(http.MethodPost, "/api/v1/placement/session", nil)
	state := decode(t, resp)
	resp.Body.Close()
	qid := state["nextQuestion"].(map[string]any)["id"].(string)

	resp = c.do(http.MethodPost, "/api/v1/placement/session/answers", map[string]string{
		"questionId": qid, "answer": "ok",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp = c.do(http.MethodPost, "/api/v1/placement/session/answers", map[string]string{
		"questionId": qid, "answer": "ok",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode, "no question repeats per session")
}

func TestPlacementInvalidUUID400(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	c.register("ana@example.com", "Ana")
	c.do(http.MethodPost, "/api/v1/placement/session", nil).Body.Close()

	resp := c.do(http.MethodPost, "/api/v1/placement/session/answers", map[string]string{
		"questionId": "not-a-uuid", "answer": "ok",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func jsonString(m map[string]any, key string) *string {
	if v, ok := m[key].(string); ok {
		return &v
	}
	return nil
}
