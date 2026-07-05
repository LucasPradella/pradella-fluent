// Package seed loads the MVP content (20 lessons + placement bank) into an
// empty database. Idempotent: skips seeding when content already exists.
package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
)

//go:embed placement_questions.json
var placementJSON []byte

//go:embed lessons.json
var lessonsJSON []byte

type seedQuestion struct {
	Band     string   `json:"band"`
	Type     string   `json:"type"`
	Prompt   string   `json:"prompt"`
	Options  []string `json:"options"`
	Correct  string   `json:"correct"`
	AudioURL string   `json:"audioUrl"`
}

type seedExercise struct {
	Type     string   `json:"type"`
	Prompt   string   `json:"prompt"`
	Target   string   `json:"target"`
	Options  []string `json:"options"`
	AudioURL string   `json:"audioUrl"`
	Order    int      `json:"order"`
}

type seedLesson struct {
	Title     string         `json:"title"`
	Focus     string         `json:"focus"`
	XP        int            `json:"xp"`
	Order     int            `json:"order"`
	Exercises []seedExercise `json:"exercises"`
}

type seedModule struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Theme       string       `json:"theme"`
	Level       string       `json:"level"`
	Order       int          `json:"order"`
	Lessons     []seedLesson `json:"lessons"`
}

// Load seeds placement questions and lesson content when absent.
func Load(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	q := gen.New(pool)

	nQuestions, err := q.CountPlacementQuestions(ctx)
	if err != nil {
		return fmt.Errorf("seed: count questions: %w", err)
	}
	if nQuestions == 0 {
		if err := loadQuestions(ctx, q); err != nil {
			return err
		}
		logger.Info("seeded placement question bank")
	}

	nModules, err := q.CountModules(ctx)
	if err != nil {
		return fmt.Errorf("seed: count modules: %w", err)
	}
	if nModules == 0 {
		if err := loadContent(ctx, q); err != nil {
			return err
		}
		logger.Info("seeded lesson content")
	}
	return nil
}

func loadQuestions(ctx context.Context, q *gen.Queries) error {
	var questions []seedQuestion
	if err := json.Unmarshal(placementJSON, &questions); err != nil {
		return fmt.Errorf("seed: parse placement_questions.json: %w", err)
	}
	for i, sq := range questions {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		opts, err := json.Marshal(sq.Options)
		if err != nil {
			return err
		}
		audio := pgText(sq.AudioURL)
		if err := q.InsertPlacementQuestion(ctx, gen.InsertPlacementQuestionParams{
			ID:            id,
			CefrBand:      gen.CefrBand(sq.Band),
			QuestionType:  gen.PlacementQuestionType(sq.Type),
			Prompt:        sq.Prompt,
			Options:       opts,
			CorrectOption: sq.Correct,
			AudioAssetUrl: audio,
		}); err != nil {
			return fmt.Errorf("seed: question %d: %w", i, err)
		}
	}
	return nil
}

func loadContent(ctx context.Context, q *gen.Queries) error {
	var modules []seedModule
	if err := json.Unmarshal(lessonsJSON, &modules); err != nil {
		return fmt.Errorf("seed: parse lessons.json: %w", err)
	}
	for _, m := range modules {
		moduleID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		if err := q.InsertModule(ctx, gen.InsertModuleParams{
			ID:              moduleID,
			Title:           m.Title,
			Description:     m.Description,
			ThemeType:       gen.ThemeType(m.Theme),
			DifficultyLevel: gen.ProficiencyLevel(m.Level),
			SequentialOrder: int32(m.Order), //nolint:gosec // seed data
		}); err != nil {
			return fmt.Errorf("seed: module %q: %w", m.Title, err)
		}
		for _, l := range m.Lessons {
			lessonID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := q.InsertLesson(ctx, gen.InsertLessonParams{
				ID:               lessonID,
				ModuleID:         moduleID,
				Title:            l.Title,
				PedagogicalFocus: l.Focus,
				XpReward:         int32(l.XP),    //nolint:gosec // seed data
				SequentialOrder:  int32(l.Order), //nolint:gosec // seed data
			}); err != nil {
				return fmt.Errorf("seed: lesson %q: %w", l.Title, err)
			}
			for _, e := range l.Exercises {
				exerciseID, err := uuid.NewV7()
				if err != nil {
					return err
				}
				var opts []byte
				if e.Options != nil {
					opts, err = json.Marshal(e.Options)
					if err != nil {
						return err
					}
				}
				if err := q.InsertExercise(ctx, gen.InsertExerciseParams{
					ID:               exerciseID,
					LessonID:         lessonID,
					ExerciseType:     gen.ExerciseType(e.Type),
					PromptContext:    e.Prompt,
					TargetAnswerText: e.Target,
					Options:          opts,
					AudioAssetUrl:    pgText(e.AudioURL),
					SequentialOrder:  int32(e.Order), //nolint:gosec // seed data
				}); err != nil {
					return fmt.Errorf("seed: exercise %d of %q: %w", e.Order, l.Title, err)
				}
			}
		}
	}
	return nil
}
