// Package user holds the learner account entity.
package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// User is a learner account. Level is empty until placement completes.
type User struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string // empty for OAuth-only accounts
	DisplayName   string
	Level         domain.Level // "" == not placed yet
	CurrentStreak int
	LongestStreak int
	Timezone      string // IANA name; drives day bucketing
	CreatedAt     time.Time
}

// Location resolves the user's IANA timezone, falling back to UTC only if
// the stored name is invalid (should not happen — validated on write).
func (u User) Location() *time.Location {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// Identity links an external or local credential to a user (FR-001).
type Identity struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Provider string // github | google | email
	Subject  string // provider's stable id (or email for provider=email)
}
