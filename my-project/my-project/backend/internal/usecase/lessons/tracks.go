package lessons

import (
	"context"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// LessonSummary is a lesson row in the track listing.
type LessonSummary struct {
	ID        uuid.UUID
	Title     string
	XPReward  int
	Completed bool
}

// ModuleDTO is a module with lock state for the current user (FR-006).
type ModuleDTO struct {
	ID         uuid.UUID
	Title      string
	Descr      string
	Theme      string
	Difficulty domain.Level
	Locked     bool
	Lessons    []LessonSummary
}

// Service wires the lessons use cases.
type Service struct {
	content  ContentRepo
	progress ProgressRepo
	users    UserReader
}

func New(content ContentRepo, progress ProgressRepo, users UserReader) *Service {
	return &Service{content: content, progress: progress, users: users}
}

// Tracks lists all modules grouped with their lessons, flagging lock state
// against the user's proficiency level and per-lesson completion.
func (s *Service) Tracks(ctx context.Context, userID uuid.UUID) ([]ModuleDTO, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	modules, err := s.content.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	lessonRows, err := s.content.ListLessons(ctx)
	if err != nil {
		return nil, err
	}
	completedIDs, err := s.content.CompletedLessonIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	completed := make(map[uuid.UUID]bool, len(completedIDs))
	for _, id := range completedIDs {
		completed[id] = true
	}

	byModule := make(map[uuid.UUID][]LessonSummary)
	for _, l := range lessonRows {
		byModule[l.ModuleID] = append(byModule[l.ModuleID], LessonSummary{
			ID:        l.ID,
			Title:     l.Title,
			XPReward:  l.XP,
			Completed: completed[l.ID],
		})
	}

	out := make([]ModuleDTO, 0, len(modules))
	for _, m := range modules {
		out = append(out, ModuleDTO{
			ID:         m.ID,
			Title:      m.Title,
			Descr:      m.Description,
			Theme:      m.Theme,
			Difficulty: m.Difficulty,
			Locked:     m.LockedFor(u.Level),
			Lessons:    byModule[m.ID],
		})
	}
	return out, nil
}
