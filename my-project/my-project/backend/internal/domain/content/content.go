// Package content holds the learning-content entities (FR-006..FR-012).
package content

import (
	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// ExerciseType enumerates the supported exercise mechanics.
type ExerciseType string

const (
	Translate       ExerciseType = "translate"
	FillBlank       ExerciseType = "fill_blank"
	ListeningChoice ExerciseType = "listening_choice"
	ListeningOrder  ExerciseType = "listening_order"
	Speaking        ExerciseType = "speaking"
)

// Module groups lessons under a theme and difficulty level.
type Module struct {
	ID          uuid.UUID
	Title       string
	Description string
	Theme       string // travel | tech
	Difficulty  domain.Level
	Order       int
}

// LockedFor reports whether the module is gated for a learner level (FR-006).
func (m Module) LockedFor(level domain.Level) bool {
	return !level.Allows(m.Difficulty)
}

// Lesson is a task-framed unit inside a module (FR-008).
type Lesson struct {
	ID       uuid.UUID
	ModuleID uuid.UUID
	Title    string
	Focus    string // pedagogical/task framing
	XP       int
	Order    int
}

// Exercise is one interactive step of a lesson. Target is never exposed to
// clients except as the sentence to read aloud in speaking exercises.
type Exercise struct {
	ID       uuid.UUID
	LessonID uuid.UUID
	Type     ExerciseType
	Prompt   string
	Target   string
	Options  []string // choices or word blocks for listening types
	AudioURL string   // listening types only
	Order    int
}
