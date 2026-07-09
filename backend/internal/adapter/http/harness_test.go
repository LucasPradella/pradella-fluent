package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpapi "github.com/pradella/fluentdev/backend/internal/adapter/http"
	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/placement"
	"github.com/pradella/fluentdev/backend/internal/infra/transcriber"
	"github.com/pradella/fluentdev/backend/internal/testutil"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
	speechuc "github.com/pradella/fluentdev/backend/internal/usecase/speech"
)

// stubTranscriber lets tests script transcription outcomes.
type stubTranscriber struct {
	mu   sync.Mutex
	text string
	err  error
}

func (s *stubTranscriber) set(text string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text, s.err = text, err
}

func (s *stubTranscriber) Transcribe(context.Context, io.Reader, transcriber.TranscribeOpts) (transcriber.Transcript, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return transcriber.Transcript{}, s.err
	}
	return transcriber.Transcript{Text: s.text, Provider: "stub"}, nil
}

// harness bundles the running test server with its fakes.
type harness struct {
	srv         *httptest.Server
	users       *testutil.MemUserRepo
	progress    *testutil.MemProgressRepo
	contentRepo *testutil.MemContentRepo
	placement   *testutil.MemPlacementRepo
	transcriber *stubTranscriber
	logBuf      *bytes.Buffer

	// content fixture handles
	basicModule    content.Module
	advancedModule content.Module
	basicLesson    content.Lesson
	advancedLesson content.Lesson
	writingEx      content.Exercise
	speakingEx     content.Exercise
	advancedEx     content.Exercise
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	users := testutil.NewMemUserRepo()
	sessions := testutil.NewMemSessionRepo()
	progress := testutil.NewMemProgressRepo()

	// Placement bank: 8 questions per band, all with correct answer "ok".
	var questions []placementuc.Question
	for _, band := range []placement.Band{placement.BandA1, placement.BandA2, placement.BandB1, placement.BandB2, placement.BandC1} {
		for i := 0; i < 8; i++ {
			questions = append(questions, placementuc.Question{
				ID: uuid.New(), Band: band, Type: "choice",
				Prompt:  fmt.Sprintf("%s question %d", band, i),
				Options: []string{"ok", "no", "maybe"}, Correct: "ok",
			})
		}
	}
	placementRepo := testutil.NewMemPlacementRepo(questions)

	// Content fixture: one basic module (writing + speaking) and one
	// advanced module (locked for basic users).
	basicModule := content.Module{ID: uuid.New(), Title: "Básico", Theme: "travel", Difficulty: domain.LevelBasic, Order: 1}
	advancedModule := content.Module{ID: uuid.New(), Title: "Avançado", Theme: "tech", Difficulty: domain.LevelAdvanced, Order: 1}
	basicLesson := content.Lesson{ID: uuid.New(), ModuleID: basicModule.ID, Title: "Check-in", XP: 50, Order: 1}
	advancedLesson := content.Lesson{ID: uuid.New(), ModuleID: advancedModule.ID, Title: "Arquitetura", XP: 90, Order: 1}
	writingEx := content.Exercise{
		ID: uuid.New(), LessonID: basicLesson.ID, Type: content.Translate,
		Prompt: "Traduza", Target: "I have one bag to check", Order: 1,
	}
	speakingEx := content.Exercise{
		ID: uuid.New(), LessonID: basicLesson.ID, Type: content.Speaking,
		Prompt: "Leia em voz alta", Target: "I would like a window seat please", Order: 2,
	}
	advancedEx := content.Exercise{
		ID: uuid.New(), LessonID: advancedLesson.ID, Type: content.Translate,
		Prompt: "Traduza", Target: "we opted for eventual consistency", Order: 1,
	}
	contentRepo := &testutil.MemContentRepo{
		Modules:   []content.Module{basicModule, advancedModule},
		Lessons:   []content.Lesson{basicLesson, advancedLesson},
		Exercises: []content.Exercise{writingEx, speakingEx, advancedEx},
		Progress:  progress,
	}
	progress.Content = contentRepo
	progress.Users = users

	stub := &stubTranscriber{text: "I would like a window seat please"}

	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))

	authSvc := auth.New(users, sessions, 30*24*time.Hour)
	router := httpapi.NewRouter(httpapi.Deps{
		Logger:        logger,
		Auth:          authSvc,
		Placement:     placementuc.New(placementRepo),
		Lessons:       lessons.New(contentRepo, progress, users),
		Speech:        speechuc.New(contentRepo, progress, users, stub),
		Dashboard:     dashboard.New(users, &testutil.MemDashboardRepo{Progress: progress, Content: contentRepo}),
		CookieName:    "fluentdev_session",
		SecureCookies: false,
		AppBaseURL:    "http://localhost:5173",
		SessionTTL:    30 * 24 * time.Hour,
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &harness{
		srv: srv, users: users, progress: progress, contentRepo: contentRepo,
		placement: placementRepo, transcriber: stub, logBuf: logBuf,
		basicModule: basicModule, advancedModule: advancedModule,
		basicLesson: basicLesson, advancedLesson: advancedLesson,
		writingEx: writingEx, speakingEx: speakingEx, advancedEx: advancedEx,
	}
}

// client is a cookie-aware API client that echoes the CSRF cookie.
type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func (h *harness) client(t *testing.T) *client {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &client{t: t, base: h.srv.URL, http: &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c *client) csrfToken() string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "fluentdev_csrf" {
			return ck.Value
		}
	}
	return ""
}

func (c *client) do(method, path string, body any) *http.Response {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	require.NoError(c.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok := c.csrfToken(); tok != "" {
		req.Header.Set("X-CSRF-Token", tok)
	}
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	return resp
}

func (c *client) doRaw(req *http.Request) *http.Response {
	c.t.Helper()
	if tok := c.csrfToken(); tok != "" {
		req.Header.Set("X-CSRF-Token", tok)
	}
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	return resp
}

// register creates an account and returns the decoded user body.
func (c *client) register(email, name string) map[string]any {
	c.t.Helper()
	resp := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": "supersecret123", "displayName": name,
	})
	defer resp.Body.Close()
	require.Equal(c.t, http.StatusCreated, resp.StatusCode)
	return decode(c.t, resp)
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

func decodeList(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	var out []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

// userID resolves the harness user id created for an e-mail.
func (h *harness) userID(t *testing.T, email string) uuid.UUID {
	u, err := h.users.GetByEmail(context.Background(), email)
	require.NoError(t, err)
	return u.ID
}
