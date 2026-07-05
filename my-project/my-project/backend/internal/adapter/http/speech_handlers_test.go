package httpapi_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/infra/transcriber"
)

// webmBytes builds a payload starting with the EBML magic (audio/webm).
func webmBytes(size int) []byte {
	b := make([]byte, size)
	copy(b, []byte{0x1A, 0x45, 0xDF, 0xA3})
	return b
}

// speechRequest builds the multipart POST for a spoken attempt.
func speechRequest(t *testing.T, c *client, exerciseID string, attemptID string, audio []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("attemptId", attemptID))
	fw, err := mw.CreateFormFile("audio", "attempt.webm")
	require.NoError(t, err)
	_, err = fw.Write(audio)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequest(http.MethodPost, c.base+"/api/v1/exercises/"+exerciseID+"/speech-attempts", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestSpeechAttemptScored201(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("I would like a window seat please", nil)

	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2048)))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := decode(t, resp)
	assert.EqualValues(t, 1, body["similarity"])
	assert.Equal(t, true, body["passed"])
	assert.NotNil(t, body["missedWords"])
}

func TestSpeechMissedWordsReported(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("I would like a seat please", nil) // omits "window"

	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2048)))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := decode(t, resp)
	missed := body["missedWords"].([]any)
	require.Len(t, missed, 1)
	assert.Equal(t, "window", missed[0])
}

func TestSpeechAudioTooLarge413(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2*1024*1024)))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestSpeechInvalidMIMESniff400(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)

	junk := bytes.Repeat([]byte{0x00}, 2048) // neither EBML nor ftyp
	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), junk))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "MIME sniffing must reject non-audio payloads")
}

func TestSpeechUnintelligible422NotScored(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("   ", nil) // provider heard nothing

	before := len(h.progress.Logs)
	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2048)))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assert.Len(t, h.progress.Logs, before, "unintelligible audio must not be scored as failure")
}

func TestSpeechProvidersDown503(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("", transcriber.ErrProvidersUnavailable)

	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2048)))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestSpeechRateLimit429(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("I would like a window seat please", nil)

	got429 := false
	for i := 0; i < 15; i++ {
		resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(1024)))
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
		}
		resp.Body.Close()
	}
	assert.True(t, got429, "speech burst must trigger the per-user rate limit (cost-bearing endpoint)")
}

func TestSpeechDuplicateReplay200(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("I would like a window seat please", nil)

	attemptID := uuid.New().String()
	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), attemptID, webmBytes(2048)))
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	first := decode(t, resp)
	resp.Body.Close()

	resp = c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), attemptID, webmBytes(2048)))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "replayed speech attempt must be 200")
	second := decode(t, resp)
	assert.Equal(t, first["similarity"], second["similarity"])
	assert.Len(t, h.progress.Logs, 1)
}

func TestSpeechFailureFeedsReviewQueue(t *testing.T) {
	h := newHarness(t)
	c := h.client(t)
	uid := placeUser(t, h, c, "dev@example.com", domain.LevelBasic)
	h.transcriber.set("something completely different entirely", nil)

	resp := c.doRaw(speechRequest(t, c, h.speakingEx.ID.String(), uuid.New().String(), webmBytes(2048)))
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, false, body["passed"])

	_, err := h.progress.GetReviewItem(t.Context(), uid, h.speakingEx.ID)
	assert.NoError(t, err, "failed speaking attempt must enter the review queue")
}
