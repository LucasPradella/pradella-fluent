package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// placeUser registers and places the user at the given level.
func placeUser(t *testing.T, h *harness, c *client, email string, level domain.Level) uuid.UUID {
	t.Helper()
	c.register(email, "User")
	uid := h.userID(t, email)
	h.users.SetLevel(uid, level)
	return uid
}

func TestTracksLockState(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.do(http.MethodGet, "/api/v1/tracks", nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	modules := decodeList(t, resp)
	require.Len(t, modules, 2)

	byTitle := map[string]map[string]any{}
	for _, m := range modules {
		byTitle[m["title"].(string)] = m
	}
	assert.False(t, byTitle["Básico"]["locked"].(bool))
	assert.True(t, byTitle["Avançado"]["locked"].(bool), "above-level track must be locked (FR-006)")
	lessons := byTitle["Básico"]["lessons"].([]any)
	require.NotEmpty(t, lessons)
	first := lessons[0].(map[string]any)
	assert.Equal(t, false, first["completed"])
	assert.EqualValues(t, 50, first["xpReward"])
}

func TestLockedLesson403(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.do(http.MethodGet, "/api/v1/lessons/"+h.advancedLesson.ID.String(), nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))
}

func TestLessonExcludesCorrectAnswers(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.do(http.MethodGet, "/api/v1/lessons/"+h.basicLesson.ID.String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)

	assert.NotContains(t, string(raw), "I have one bag to check",
		"writing exercise target must not be exposed")

	var lesson map[string]any
	require.NoError(t, json.Unmarshal(raw, &lesson))
	exercises := lesson["exercises"].([]any)
	require.Len(t, exercises, 2)

	for _, raw := range exercises {
		ex := raw.(map[string]any)
		if ex["exerciseType"] == "speaking" {
			assert.Equal(t, "I would like a window seat please", *jsonString(ex, "targetSentence"),
				"speaking exercises expose the sentence to read aloud")
		} else {
			assert.Nil(t, ex["targetSentence"])
		}
	}
}

func TestAttempt201AndDuplicate200(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	attemptID := uuid.New().String()
	payload := map[string]any{"attemptId": attemptID, "answer": "I have one bag to check"}

	resp := c.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts", payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	first := decode(t, resp)
	resp.Body.Close()
	assert.Equal(t, true, first["correct"])

	// Same attemptId replayed (outbox) → 200 with the same body.
	resp = c.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts", payload)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "duplicate replay must be 200, not 201")
	second := decode(t, resp)
	assert.Equal(t, first["correct"], second["correct"])
}

func TestAttemptWrongAnswerShowsExpected(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.do(http.MethodPost, "/api/v1/exercises/"+h.writingEx.ID.String()+"/attempts",
		map[string]any{"attemptId": uuid.New().String(), "answer": "completely wrong sentence here"})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := decode(t, resp)
	assert.Equal(t, false, body["correct"])
	assert.Equal(t, "I have one bag to check", *jsonString(body, "expectedAnswer"),
		"wrong answers must reveal the expected answer (US2 scenario 3)")
}

func TestAttemptOnLockedTrack403(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.do(http.MethodPost, "/api/v1/exercises/"+h.advancedEx.ID.String()+"/attempts",
		map[string]any{"attemptId": uuid.New().String(), "answer": "anything"})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
