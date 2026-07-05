package transcriber_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/infra/transcriber"
)

func fakeProvider(t *testing.T, status int, body string, delay time.Duration) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/audio/transcriptions", r.URL.Path)
		require.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func audio() *strings.Reader { return strings.NewReader("fake-webm-bytes") }

func TestGroqSuccess(t *testing.T) {
	srv := fakeProvider(t, 200, `{"text":"hello world"}`, 0)
	groq := transcriber.NewGroq(srv.URL, "key", nil)

	out, err := groq.Transcribe(context.Background(), audio(), transcriber.TranscribeOpts{MIMEType: "audio/webm"})
	require.NoError(t, err)
	assert.Equal(t, "hello world", out.Text)
	assert.Equal(t, "groq", out.Provider)
}

func TestFailoverOnPrimary429(t *testing.T) {
	primary := fakeProvider(t, 429, `{"error":"rate limited"}`, 0)
	secondary := fakeProvider(t, 200, `{"text":"from secondary"}`, 0)

	f := transcriber.NewFailover(
		transcriber.NewGroq(primary.URL, "key", nil),
		transcriber.NewOpenAI(secondary.URL, "key", nil),
		nil,
	)
	out, err := f.Transcribe(context.Background(), audio(), transcriber.TranscribeOpts{MIMEType: "audio/webm"})
	require.NoError(t, err)
	assert.Equal(t, "from secondary", out.Text)
	assert.Equal(t, "openai", out.Provider)
}

func TestFailoverOnPrimaryTimeout(t *testing.T) {
	// Primary exceeds the 2.5 s soft deadline; secondary answers fast.
	primary := fakeProvider(t, 200, `{"text":"too late"}`, 4*time.Second)
	secondary := fakeProvider(t, 200, `{"text":"fast answer"}`, 0)

	f := transcriber.NewFailover(
		transcriber.NewGroq(primary.URL, "key", nil),
		transcriber.NewOpenAI(secondary.URL, "key", nil),
		nil,
	)
	start := time.Now()
	out, err := f.Transcribe(context.Background(), audio(), transcriber.TranscribeOpts{MIMEType: "audio/webm"})
	require.NoError(t, err)
	assert.Equal(t, "fast answer", out.Text)
	assert.Less(t, time.Since(start), 4*time.Second, "must not wait for the slow primary")
}

func TestBothProvidersDown(t *testing.T) {
	primary := fakeProvider(t, 500, `{"error":"boom"}`, 0)
	secondary := fakeProvider(t, 503, `{"error":"down"}`, 0)

	f := transcriber.NewFailover(
		transcriber.NewGroq(primary.URL, "key", nil),
		transcriber.NewOpenAI(secondary.URL, "key", nil),
		nil,
	)
	_, err := f.Transcribe(context.Background(), audio(), transcriber.TranscribeOpts{MIMEType: "audio/webm"})
	assert.ErrorIs(t, err, transcriber.ErrProvidersUnavailable)
}

func TestNoSecondaryConfigured(t *testing.T) {
	primary := fakeProvider(t, 500, `{"error":"boom"}`, 0)
	f := transcriber.NewFailover(transcriber.NewGroq(primary.URL, "key", nil), nil, nil)
	_, err := f.Transcribe(context.Background(), audio(), transcriber.TranscribeOpts{})
	assert.ErrorIs(t, err, transcriber.ErrProvidersUnavailable)
}
