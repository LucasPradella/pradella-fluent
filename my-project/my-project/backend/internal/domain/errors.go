// Package domain holds entities and business rules shared by all
// bounded contexts. It has zero external dependencies.
package domain

import "errors"

var (
	// ErrNotFound signals that a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrForbidden signals that the acting user may not access the entity.
	ErrForbidden = errors.New("forbidden")
	// ErrConflict signals a uniqueness or state-transition violation.
	ErrConflict = errors.New("conflict")
	// ErrInvalid signals a domain-rule validation failure.
	ErrInvalid = errors.New("invalid")
	// ErrUnauthorized signals a missing or invalid credential.
	ErrUnauthorized = errors.New("unauthorized")
)

// Level is a learner proficiency level (FR-005).
type Level string

const (
	LevelBasic        Level = "basic"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

// Allows reports whether a learner at level l may access content gated at
// required (FR-006). An empty level (placement not completed) allows nothing.
func (l Level) Allows(required Level) bool {
	return l.rank() >= required.rank() && l.rank() > 0
}

func (l Level) rank() int {
	switch l {
	case LevelBasic:
		return 1
	case LevelIntermediate:
		return 2
	case LevelAdvanced:
		return 3
	default:
		return 0
	}
}
