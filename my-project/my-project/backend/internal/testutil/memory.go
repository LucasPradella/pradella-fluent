// Package testutil provides in-memory implementations of the usecase ports
// for unit and contract tests (no database required).
package testutil

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
	"github.com/pradella/fluentdev/backend/internal/domain/placement"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
)

// ─── Users ────────────────────────────────────────────────────────────────

type MemUserRepo struct {
	mu         sync.Mutex
	Users      map[uuid.UUID]user.User
	Identities []user.Identity
}

func NewMemUserRepo() *MemUserRepo {
	return &MemUserRepo{Users: map[uuid.UUID]user.User{}}
}

func (r *MemUserRepo) Create(_ context.Context, u user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.Users {
		if strings.EqualFold(e.Email, u.Email) {
			return domain.ErrConflict
		}
	}
	r.Users[u.ID] = u
	return nil
}

func (r *MemUserRepo) GetByEmail(_ context.Context, email string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.Users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return user.User{}, domain.ErrNotFound
}

func (r *MemUserRepo) GetByID(_ context.Context, id uuid.UUID) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.Users[id]
	if !ok {
		return user.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (r *MemUserRepo) CreateIdentity(_ context.Context, ident user.Identity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Identities = append(r.Identities, ident)
	return nil
}

func (r *MemUserRepo) GetIdentity(_ context.Context, provider, subject string) (user.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, i := range r.Identities {
		if i.Provider == provider && i.Subject == subject {
			return i, nil
		}
	}
	return user.Identity{}, domain.ErrNotFound
}

// SetLevel is a test helper to place a user.
func (r *MemUserRepo) SetLevel(id uuid.UUID, level domain.Level) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.Users[id]
	u.Level = level
	r.Users[id] = u
}

// ─── Sessions ─────────────────────────────────────────────────────────────

type MemSessionRepo struct {
	mu       sync.Mutex
	Sessions map[string]auth.Session // keyed by hex-ish string(hash)
}

func NewMemSessionRepo() *MemSessionRepo {
	return &MemSessionRepo{Sessions: map[string]auth.Session{}}
}

func (r *MemSessionRepo) Create(_ context.Context, s auth.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Sessions[string(s.TokenHash)] = s
	return nil
}

func (r *MemSessionRepo) GetByTokenHash(_ context.Context, hash []byte) (auth.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.Sessions[string(hash)]
	if !ok || time.Now().After(s.ExpiresAt) {
		return auth.Session{}, domain.ErrNotFound
	}
	return s, nil
}

func (r *MemSessionRepo) Touch(_ context.Context, id uuid.UUID, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.Sessions {
		if s.ID == id {
			s.ExpiresAt = expiresAt
			r.Sessions[k] = s
		}
	}
	return nil
}

func (r *MemSessionRepo) DeleteByTokenHash(_ context.Context, hash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Sessions, string(hash))
	return nil
}

// ─── Placement ────────────────────────────────────────────────────────────

type MemPlacementRepo struct {
	mu        sync.Mutex
	Sessions  map[uuid.UUID]placementuc.Session // by session id
	Answers   map[uuid.UUID][]placementuc.Answer
	Questions []placementuc.Question
	// LevelAssigned records CompleteAndAssignLevel calls (userID → level).
	LevelAssigned map[uuid.UUID]domain.Level
}

func NewMemPlacementRepo(questions []placementuc.Question) *MemPlacementRepo {
	return &MemPlacementRepo{
		Sessions:      map[uuid.UUID]placementuc.Session{},
		Answers:       map[uuid.UUID][]placementuc.Answer{},
		Questions:     questions,
		LevelAssigned: map[uuid.UUID]domain.Level{},
	}
}

func (r *MemPlacementRepo) GetActiveSession(_ context.Context, userID uuid.UUID) (placementuc.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.Sessions {
		if s.UserID == userID && s.Status == "active" {
			return s, nil
		}
	}
	return placementuc.Session{}, domain.ErrNotFound
}

func (r *MemPlacementRepo) CreateSession(_ context.Context, id, userID uuid.UUID) (placementuc.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := placementuc.Session{
		ID: id, UserID: userID, Status: "active",
		CurrentBand: placement.StartBand, QuestionsServed: 0,
	}
	r.Sessions[id] = s
	return s, nil
}

func (r *MemPlacementRepo) AbandonActiveSessions(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, s := range r.Sessions {
		if s.UserID == userID && s.Status == "active" {
			s.Status = "abandoned"
			r.Sessions[id] = s
		}
	}
	return nil
}

func (r *MemPlacementRepo) UpdateProgress(_ context.Context, sessionID uuid.UUID, band placement.Band, served int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.Sessions[sessionID]
	s.CurrentBand = band
	s.QuestionsServed = served
	r.Sessions[sessionID] = s
	return nil
}

func (r *MemPlacementRepo) CompleteAndAssignLevel(_ context.Context, sessionID, userID uuid.UUID, level domain.Level) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.Sessions[sessionID]
	s.Status = "completed"
	s.AssignedLevel = level
	r.Sessions[sessionID] = s
	r.LevelAssigned[userID] = level
	return nil
}

func (r *MemPlacementRepo) InsertAnswer(_ context.Context, id, sessionID, questionID uuid.UUID, testlet int, correct bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.Answers[sessionID] {
		if a.QuestionID == questionID {
			return domain.ErrConflict
		}
	}
	r.Answers[sessionID] = append(r.Answers[sessionID], placementuc.Answer{
		QuestionID: questionID, TestletIndex: testlet, IsCorrect: correct, AnsweredAt: time.Now(),
	})
	return nil
}

func (r *MemPlacementRepo) ListAnswers(_ context.Context, sessionID uuid.UUID) ([]placementuc.Answer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]placementuc.Answer(nil), r.Answers[sessionID]...), nil
}

func (r *MemPlacementRepo) GetQuestion(_ context.Context, id uuid.UUID) (placementuc.Question, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, q := range r.Questions {
		if q.ID == id {
			return q, nil
		}
	}
	return placementuc.Question{}, domain.ErrNotFound
}

func (r *MemPlacementRepo) PickUnservedQuestion(_ context.Context, band placement.Band, sessionID uuid.UUID) (placementuc.Question, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	answered := map[uuid.UUID]bool{}
	for _, a := range r.Answers[sessionID] {
		answered[a.QuestionID] = true
	}
	for _, q := range r.Questions {
		if q.Band == band && !answered[q.ID] {
			return q, nil
		}
	}
	return placementuc.Question{}, domain.ErrNotFound
}

// ─── Content ──────────────────────────────────────────────────────────────

type MemContentRepo struct {
	Modules   []content.Module
	Lessons   []content.Lesson
	Exercises []content.Exercise
	Progress  *MemProgressRepo // for CompletedLessonIDs
}

func (r *MemContentRepo) ListModules(context.Context) ([]content.Module, error) {
	return r.Modules, nil
}

func (r *MemContentRepo) ListLessons(context.Context) ([]content.Lesson, error) {
	return r.Lessons, nil
}

func (r *MemContentRepo) CompletedLessonIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for _, l := range r.Lessons {
		all := true
		any := false
		for _, e := range r.Exercises {
			if e.LessonID != l.ID {
				continue
			}
			any = true
			if r.Progress == nil || !r.Progress.passed(userID, e.ID) {
				all = false
				break
			}
		}
		if any && all {
			out = append(out, l.ID)
		}
	}
	return out, nil
}

func (r *MemContentRepo) GetLessonWithModule(_ context.Context, lessonID uuid.UUID) (content.Lesson, content.Module, error) {
	for _, l := range r.Lessons {
		if l.ID == lessonID {
			for _, m := range r.Modules {
				if m.ID == l.ModuleID {
					return l, m, nil
				}
			}
		}
	}
	return content.Lesson{}, content.Module{}, domain.ErrNotFound
}

func (r *MemContentRepo) ListExercises(_ context.Context, lessonID uuid.UUID) ([]content.Exercise, error) {
	var out []content.Exercise
	for _, e := range r.Exercises {
		if e.LessonID == lessonID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out, nil
}

func (r *MemContentRepo) GetExerciseContext(_ context.Context, exerciseID uuid.UUID) (content.Exercise, content.Lesson, content.Module, error) {
	for _, e := range r.Exercises {
		if e.ID == exerciseID {
			for _, l := range r.Lessons {
				if l.ID == e.LessonID {
					for _, m := range r.Modules {
						if m.ID == l.ModuleID {
							return e, l, m, nil
						}
					}
				}
			}
		}
	}
	return content.Exercise{}, content.Lesson{}, content.Module{}, domain.ErrNotFound
}

// ─── Progress ─────────────────────────────────────────────────────────────

type MemProgressRepo struct {
	mu      sync.Mutex
	Logs    map[uuid.UUID]lessons.LogEntry
	Reviews map[uuid.UUID]memReviewItem // by item id
	// Streaks recorded by RecordAttempt (userID → current/longest).
	StreaksByUser map[uuid.UUID][2]int
	// Content is a back-reference for lesson-scoped queries (set by tests).
	Content *MemContentRepo
	// Users mirrors the streak update the real adapter performs in the
	// same transaction (set by tests).
	Users *MemUserRepo
}

type memReviewItem struct {
	lessons.ReviewItemState
	UserID     uuid.UUID
	ExerciseID uuid.UUID
	DueAt      time.Time
	LastResult string
}

func NewMemProgressRepo() *MemProgressRepo {
	return &MemProgressRepo{
		Logs:          map[uuid.UUID]lessons.LogEntry{},
		Reviews:       map[uuid.UUID]memReviewItem{},
		StreaksByUser: map[uuid.UUID][2]int{},
	}
}

func (r *MemProgressRepo) passed(userID, exerciseID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.Logs {
		if l.UserID == userID && l.ExerciseID == exerciseID && l.Accuracy >= 0.8 {
			return true
		}
	}
	return false
}

func (r *MemProgressRepo) GetLog(_ context.Context, id uuid.UUID) (lessons.LogEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.Logs[id]
	if !ok {
		return lessons.LogEntry{}, domain.ErrNotFound
	}
	return l, nil
}

func (r *MemProgressRepo) ActivityTimestamps(_ context.Context, userID uuid.UUID) ([]time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []time.Time
	for _, l := range r.Logs {
		if l.UserID == userID {
			out = append(out, l.CompletedAt)
		}
	}
	return out, nil
}

func (r *MemProgressRepo) CountUnpassedInLesson(_ context.Context, lessonID, userID uuid.UUID) (int, error) {
	// The fake needs exercise → lesson mapping; tests wire Content here.
	if r.Content == nil {
		return 0, nil
	}
	n := 0
	for _, e := range r.Content.Exercises {
		if e.LessonID == lessonID && !r.passed(userID, e.ID) {
			n++
		}
	}
	return n, nil
}

func (r *MemProgressRepo) IsExercisePassed(_ context.Context, userID, exerciseID uuid.UUID) (bool, error) {
	return r.passed(userID, exerciseID), nil
}

func (r *MemProgressRepo) HasLessonXPAward(_ context.Context, userID, lessonID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.Logs {
		if l.UserID == userID && strings.Contains(string(l.Detail), lessonID.String()) &&
			strings.Contains(string(l.Detail), "xpLessonId") {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemProgressRepo) RecordAttempt(_ context.Context, p lessons.RecordParams) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.Logs[p.Log.ID]; exists {
		return false, nil
	}
	r.Logs[p.Log.ID] = p.Log
	r.StreaksByUser[p.Log.UserID] = [2]int{p.CurrentStreak, p.LongestStreak}
	if r.Users != nil {
		r.Users.mu.Lock()
		if u, ok := r.Users.Users[p.Log.UserID]; ok {
			u.CurrentStreak = p.CurrentStreak
			u.LongestStreak = p.LongestStreak
			r.Users.Users[p.Log.UserID] = u
		}
		r.Users.mu.Unlock()
	}

	if p.Review != nil {
		rc := p.Review
		switch {
		case rc.Delete:
			delete(r.Reviews, rc.ItemID)
		default:
			r.Reviews[rc.ItemID] = memReviewItem{
				ReviewItemState: lessons.ReviewItemState{
					ItemID:       rc.ItemID,
					IntervalDays: rc.IntervalDays,
					StreakAt7d:   rc.StreakAt7d,
					FailureCount: rc.FailureCount,
				},
				UserID:     p.Log.UserID,
				ExerciseID: rc.ExerciseID,
				DueAt:      rc.DueAt,
				LastResult: rc.LastResult,
			}
		}
	}
	return true, nil
}

// SeedReview inserts a review item directly (test setup helper).
func (r *MemProgressRepo) SeedReview(itemID, userID, exerciseID uuid.UUID, dueAt time.Time, failureCount int) {
	r.SeedReviewState(itemID, userID, exerciseID, dueAt, lessons.ReviewItemState{
		ItemID: itemID, IntervalDays: 1, FailureCount: failureCount,
	})
}

// SeedReviewState inserts a review item with full scheduling state.
func (r *MemProgressRepo) SeedReviewState(itemID, userID, exerciseID uuid.UUID, dueAt time.Time, state lessons.ReviewItemState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.ItemID = itemID
	r.Reviews[itemID] = memReviewItem{
		ReviewItemState: state,
		UserID:          userID, ExerciseID: exerciseID, DueAt: dueAt, LastResult: "failed",
	}
}

// HasReviewItemID reports whether an item id is still queued.
func (r *MemProgressRepo) HasReviewItemID(itemID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.Reviews[itemID]
	return ok
}

func (r *MemProgressRepo) GetReviewItem(_ context.Context, userID, exerciseID uuid.UUID) (lessons.ReviewItemState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range r.Reviews {
		if it.UserID == userID && it.ExerciseID == exerciseID {
			return it.ReviewItemState, nil
		}
	}
	return lessons.ReviewItemState{}, domain.ErrNotFound
}

// Content is an optional back-reference so lesson-scoped counts work.
var _ lessons.ProgressRepo = (*MemProgressRepo)(nil)

// ─── Dashboard ────────────────────────────────────────────────────────────

type MemDashboardRepo struct {
	Progress *MemProgressRepo
	Content  *MemContentRepo
}

func (r *MemDashboardRepo) HeatmapCounts(_ context.Context, userID uuid.UUID, tz string) (map[string]int, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	out := map[string]int{}
	r.Progress.mu.Lock()
	defer r.Progress.mu.Unlock()
	for _, l := range r.Progress.Logs {
		if l.UserID == userID {
			out[l.CompletedAt.In(loc).Format("2006-01-02")]++
		}
	}
	return out, nil
}

func (r *MemDashboardRepo) TotalXP(_ context.Context, userID uuid.UUID) (int, error) {
	r.Progress.mu.Lock()
	defer r.Progress.mu.Unlock()
	total := 0
	for _, l := range r.Progress.Logs {
		if l.UserID != userID {
			continue
		}
		total += extractXP(l.Detail)
	}
	return total, nil
}

func (r *MemDashboardRepo) CountDueReviews(_ context.Context, userID uuid.UUID) (int, error) {
	items, _ := r.ListDueReviews(context.Background(), userID)
	return len(items), nil
}

func (r *MemDashboardRepo) ListDueReviews(_ context.Context, userID uuid.UUID) ([]dashboard.DueReview, error) {
	r.Progress.mu.Lock()
	defer r.Progress.mu.Unlock()
	var out []dashboard.DueReview
	now := time.Now()
	for _, it := range r.Progress.Reviews {
		if it.UserID != userID || it.DueAt.After(now) {
			continue
		}
		var ex content.Exercise
		if r.Content != nil {
			for _, e := range r.Content.Exercises {
				if e.ID == it.ExerciseID {
					ex = e
					break
				}
			}
		}
		out = append(out, dashboard.DueReview{
			ID: it.ItemID, DueAt: it.DueAt, FailureCount: it.FailureCount, Exercise: ex,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueAt.Before(out[j].DueAt) })
	return out, nil
}

func extractXP(detail []byte) int {
	s := string(detail)
	idx := strings.Index(s, `"xpAwarded":`)
	if idx < 0 {
		return 0
	}
	n := 0
	for _, c := range s[idx+len(`"xpAwarded":`):] {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
			continue
		}
		break
	}
	return n
}
