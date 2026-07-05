// Package speechuc handles spoken attempts: audio validation, transcription
// with failover, similarity scoring and progress logging (US3, FR-013..017).
package speechuc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
	domainspeech "github.com/pradella/fluentdev/backend/internal/domain/speech"
	"github.com/pradella/fluentdev/backend/internal/infra/transcriber"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

// MaxAudioBytes caps uploads at 1.5 MB (~30 s of opus — FR-013, OWASP A03).
const MaxAudioBytes = 1536 * 1024

// ErrUnintelligible marks audio the provider could not turn into words —
// surfaced as 422 and NOT logged as a failed attempt (US3 edge case).
var ErrUnintelligible = errors.New("audio unintelligible")

// ErrProvidersUnavailable re-exports the transcriber sentinel for handlers.
var ErrProvidersUnavailable = transcriber.ErrProvidersUnavailable

// Result is the scored speaking outcome (SpeechResult in the contract).
type Result struct {
	Similarity  float64  `json:"similarity"`
	Passed      bool     `json:"passed"`
	Transcript  string   `json:"transcript"`
	MissedWords []string `json:"missedWords"`
	XPAwarded   int      `json:"xpAwarded"`
	Duplicate   bool     `json:"-"`
}

// speechDetail is persisted in progress_logs.detail (no audio, no PII).
type speechDetail struct {
	Kind        string   `json:"kind"` // always "speaking"
	Similarity  float64  `json:"similarity"`
	Passed      bool     `json:"passed"`
	Transcript  string   `json:"transcript"`
	MissedWords []string `json:"missedWords,omitempty"`
	Provider    string   `json:"provider"`
	XPAwarded   int      `json:"xpAwarded,omitempty"`
	XPLessonID  string   `json:"xpLessonId,omitempty"`
}

// Service wires the speech attempt use case; it reuses the lessons ports for
// content, progress and users so review feeding shares one code path (T058).
type Service struct {
	lessons     *lessons.Service // reused helpers via exported methods
	content     lessons.ContentRepo
	progress    lessons.ProgressRepo
	users       lessons.UserReader
	transcriber transcriber.Transcriber
}

func New(content lessons.ContentRepo, progress lessons.ProgressRepo, users lessons.UserReader, t transcriber.Transcriber) *Service {
	return &Service{
		lessons:     lessons.New(content, progress, users),
		content:     content,
		progress:    progress,
		users:       users,
		transcriber: t,
	}
}

// Submit validates, transcribes and scores one spoken attempt.
func (s *Service) Submit(ctx context.Context, userID, exerciseID, attemptID uuid.UUID, audio io.Reader, declaredSize int64, isReview bool) (Result, error) {
	// Duplicate replay (offline retry of the multipart upload). Cross-user
	// attempt ids are rejected (OWASP A01).
	if prior, err := s.progress.GetLog(ctx, attemptID); err == nil {
		if prior.UserID != userID {
			return Result{}, domain.ErrForbidden
		}
		return replaySpeech(prior)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	ex, lesson, module, err := s.content.GetExerciseContext(ctx, exerciseID)
	if err != nil {
		return Result{}, err
	}
	if module.LockedFor(u.Level) {
		return Result{}, fmt.Errorf("%w: track requires level %s", domain.ErrForbidden, module.Difficulty)
	}
	if ex.Type != content.Speaking {
		return Result{}, fmt.Errorf("%w: exercise is not a speaking exercise", domain.ErrInvalid)
	}
	if declaredSize > MaxAudioBytes {
		return Result{}, fmt.Errorf("%w: audio exceeds %d bytes", domain.ErrInvalid, MaxAudioBytes)
	}

	buf, err := io.ReadAll(io.LimitReader(audio, MaxAudioBytes+1))
	if err != nil {
		return Result{}, fmt.Errorf("read audio: %w", err)
	}
	if len(buf) > MaxAudioBytes {
		return Result{}, fmt.Errorf("%w: audio exceeds %d bytes", domain.ErrInvalid, MaxAudioBytes)
	}
	mime, ok := sniffAudioMIME(buf)
	if !ok {
		return Result{}, fmt.Errorf("%w: audio must be audio/webm or audio/mp4", domain.ErrInvalid)
	}

	t, err := s.transcriber.Transcribe(ctx, bytes.NewReader(buf), transcriber.TranscribeOpts{
		MIMEType: mime,
		Language: "en",
	})
	if err != nil {
		if errors.Is(err, transcriber.ErrProvidersUnavailable) {
			return Result{}, ErrProvidersUnavailable
		}
		return Result{}, err
	}
	if strings.TrimSpace(t.Text) == "" {
		return Result{}, ErrUnintelligible // 422 — not scored as failure
	}

	score := domainspeech.ScoreTranscript(ex.Target, t.Text)
	result := Result{
		Similarity:  round4(score.Similarity),
		Passed:      score.Passed,
		Transcript:  t.Text,
		MissedWords: score.MissedWords,
	}

	// Lesson completion + one-time XP (same rules as written attempts).
	if score.Passed {
		unpassed, err := s.progress.CountUnpassedInLesson(ctx, lesson.ID, userID)
		if err != nil {
			return Result{}, err
		}
		passedBefore, err := s.progress.IsExercisePassed(ctx, userID, exerciseID)
		if err != nil {
			return Result{}, err
		}
		remaining := unpassed
		if !passedBefore {
			remaining--
		}
		if remaining <= 0 {
			awarded, err := s.progress.HasLessonXPAward(ctx, userID, lesson.ID)
			if err != nil {
				return Result{}, err
			}
			if !awarded {
				result.XPAwarded = lesson.XP
			}
		}
	}

	detail := speechDetail{
		Kind:        "speaking",
		Similarity:  result.Similarity,
		Passed:      result.Passed,
		Transcript:  t.Text,
		MissedWords: result.MissedWords,
		Provider:    t.Provider,
		XPAwarded:   result.XPAwarded,
	}
	if result.XPAwarded > 0 {
		detail.XPLessonID = lesson.ID.String()
	}
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return Result{}, err
	}

	params, err := s.lessons.BuildRecordForSpeech(ctx, u, lessons.LogEntry{
		ID:          attemptID,
		UserID:      userID,
		ExerciseID:  exerciseID,
		CompletedAt: time.Now(),
		Accuracy:    result.Similarity,
		IsReview:    isReview,
		Detail:      detailJSON,
	}, score.Passed, isReview)
	if err != nil {
		return Result{}, err
	}

	inserted, err := s.progress.RecordAttempt(ctx, params)
	if err != nil {
		return Result{}, err
	}
	if !inserted {
		if prior, err := s.progress.GetLog(ctx, attemptID); err == nil {
			return replaySpeech(prior)
		}
	}
	return result, nil
}

// sniffAudioMIME checks magic bytes: EBML (webm) or an mp4 'ftyp' box.
func sniffAudioMIME(b []byte) (string, bool) {
	if len(b) >= 4 && b[0] == 0x1A && b[1] == 0x45 && b[2] == 0xDF && b[3] == 0xA3 {
		return "audio/webm", true
	}
	if len(b) >= 12 && string(b[4:8]) == "ftyp" {
		return "audio/mp4", true
	}
	return "", false
}

func round4(f float64) float64 {
	return float64(int(f*10000+0.5)) / 10000
}

func replaySpeech(entry lessons.LogEntry) (Result, error) {
	var d speechDetail
	if err := json.Unmarshal(entry.Detail, &d); err != nil {
		return Result{}, fmt.Errorf("corrupt speech detail: %w", err)
	}
	return Result{
		Similarity:  d.Similarity,
		Passed:      d.Passed,
		Transcript:  d.Transcript,
		MissedWords: d.MissedWords,
		XPAwarded:   d.XPAwarded,
		Duplicate:   true,
	}, nil
}
