package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/content"
)

// ContentRepo implements lessons.ContentRepo.
type ContentRepo struct{ *Store }

func NewContentRepo(s *Store) *ContentRepo { return &ContentRepo{s} }

func toDomainModule(m gen.Module) content.Module {
	return content.Module{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Theme:       string(m.ThemeType),
		Difficulty:  domain.Level(m.DifficultyLevel),
		Order:       int(m.SequentialOrder),
	}
}

func toDomainLesson(l gen.Lesson) content.Lesson {
	return content.Lesson{
		ID:       l.ID,
		ModuleID: l.ModuleID,
		Title:    l.Title,
		Focus:    l.PedagogicalFocus,
		XP:       int(l.XpReward),
		Order:    int(l.SequentialOrder),
	}
}

func toDomainExercise(e gen.Exercise) content.Exercise {
	return content.Exercise{
		ID:       e.ID,
		LessonID: e.LessonID,
		Type:     content.ExerciseType(e.ExerciseType),
		Prompt:   e.PromptContext,
		Target:   e.TargetAnswerText,
		Options:  decodeOptions(e.Options),
		AudioURL: e.AudioAssetUrl.String,
		Order:    int(e.SequentialOrder),
	}
}

func (r *ContentRepo) ListModules(ctx context.Context) ([]content.Module, error) {
	rows, err := r.q.ListModules(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]content.Module, 0, len(rows))
	for _, m := range rows {
		out = append(out, toDomainModule(m))
	}
	return out, nil
}

func (r *ContentRepo) ListLessons(ctx context.Context) ([]content.Lesson, error) {
	rows, err := r.q.ListLessons(ctx)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]content.Lesson, 0, len(rows))
	for _, l := range rows {
		out = append(out, toDomainLesson(l))
	}
	return out, nil
}

func (r *ContentRepo) CompletedLessonIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q.ListCompletedLessonIDs(ctx, userID)
	return ids, mapErr(err)
}

func (r *ContentRepo) GetLessonWithModule(ctx context.Context, lessonID uuid.UUID) (content.Lesson, content.Module, error) {
	row, err := r.q.GetLessonWithModule(ctx, lessonID)
	if err != nil {
		return content.Lesson{}, content.Module{}, mapErr(err)
	}
	return toDomainLesson(row.Lesson), toDomainModule(row.Module), nil
}

func (r *ContentRepo) ListExercises(ctx context.Context, lessonID uuid.UUID) ([]content.Exercise, error) {
	rows, err := r.q.ListExercisesByLesson(ctx, lessonID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]content.Exercise, 0, len(rows))
	for _, e := range rows {
		out = append(out, toDomainExercise(e))
	}
	return out, nil
}

func (r *ContentRepo) GetExerciseContext(ctx context.Context, exerciseID uuid.UUID) (content.Exercise, content.Lesson, content.Module, error) {
	row, err := r.q.GetExerciseWithLesson(ctx, exerciseID)
	if err != nil {
		return content.Exercise{}, content.Lesson{}, content.Module{}, mapErr(err)
	}
	return toDomainExercise(row.Exercise), toDomainLesson(row.Lesson), toDomainModule(row.Module), nil
}
