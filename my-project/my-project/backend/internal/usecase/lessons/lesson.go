package lessons

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
)

// ExerciseDTO is the client-facing exercise shape. Correct answers are never
// included; TargetSentence is set only for speaking exercises (read-aloud).
type ExerciseDTO struct {
	ID             uuid.UUID
	Type           content.ExerciseType
	Prompt         string
	Options        []string
	AudioURL       string
	TargetSentence string
}

// LessonDTO is a lesson with its ordered exercises.
type LessonDTO struct {
	ID        uuid.UUID
	Title     string
	Focus     string
	XPReward  int
	Exercises []ExerciseDTO
}

// Lesson returns a lesson and its exercises, enforcing the level gate:
// a lesson in a locked track yields domain.ErrForbidden (FR-006 → 403).
func (s *Service) Lesson(ctx context.Context, userID, lessonID uuid.UUID) (LessonDTO, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return LessonDTO{}, err
	}
	lesson, module, err := s.content.GetLessonWithModule(ctx, lessonID)
	if err != nil {
		return LessonDTO{}, err
	}
	if module.LockedFor(u.Level) {
		return LessonDTO{}, fmt.Errorf("%w: track %q requires level %s", domain.ErrForbidden, module.Title, module.Difficulty)
	}
	exercises, err := s.content.ListExercises(ctx, lessonID)
	if err != nil {
		return LessonDTO{}, err
	}

	dto := LessonDTO{
		ID:        lesson.ID,
		Title:     lesson.Title,
		Focus:     lesson.Focus,
		XPReward:  lesson.XP,
		Exercises: make([]ExerciseDTO, 0, len(exercises)),
	}
	for _, ex := range exercises {
		e := ExerciseDTO{
			ID:       ex.ID,
			Type:     ex.Type,
			Prompt:   ex.Prompt,
			Options:  ex.Options,
			AudioURL: ex.AudioURL,
		}
		if ex.Type == content.Speaking {
			e.TargetSentence = ex.Target
		}
		dto.Exercises = append(dto.Exercises, e)
	}
	return dto, nil
}
