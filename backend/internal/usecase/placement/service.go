// Package placementuc orchestrates the adaptive placement test (US1):
// start/resume, server-side scoring, band walk, atomic level assignment.
package placementuc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/placement"
)

// Question is a bank question. CorrectOption never leaves the usecase.
type Question struct {
	ID       uuid.UUID
	Band     placement.Band
	Type     string // choice | listening_choice | order
	Prompt   string
	Options  []string
	Correct  string
	AudioURL string
}

// Session mirrors the persisted placement session.
type Session struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          string // active | completed | abandoned
	CurrentBand     placement.Band
	QuestionsServed int
	AssignedLevel   domain.Level // "" until completed
}

// Answer is a scored answer within a session.
type Answer struct {
	QuestionID   uuid.UUID
	TestletIndex int
	IsCorrect    bool
	AnsweredAt   time.Time
}

// Repo is the persistence port for placement flows.
type Repo interface {
	GetActiveSession(ctx context.Context, userID uuid.UUID) (Session, error)
	CreateSession(ctx context.Context, id, userID uuid.UUID) (Session, error)
	AbandonActiveSessions(ctx context.Context, userID uuid.UUID) error
	UpdateProgress(ctx context.Context, sessionID uuid.UUID, band placement.Band, served int) error
	// CompleteAndAssignLevel must set session status/assigned level and
	// users.proficiency_level in one transaction (atomic — US1).
	CompleteAndAssignLevel(ctx context.Context, sessionID, userID uuid.UUID, level domain.Level) error
	// InsertAnswer returns domain.ErrConflict when the question was already
	// answered in this session (no repeats).
	InsertAnswer(ctx context.Context, id, sessionID, questionID uuid.UUID, testlet int, correct bool) error
	ListAnswers(ctx context.Context, sessionID uuid.UUID) ([]Answer, error)
	GetQuestion(ctx context.Context, id uuid.UUID) (Question, error)
	// PickUnservedQuestion returns domain.ErrNotFound when the band bank is
	// exhausted for this session.
	PickUnservedQuestion(ctx context.Context, band placement.Band, sessionID uuid.UUID) (Question, error)
}

// NextQuestion is the client-facing question shape (no correct answer).
type NextQuestion struct {
	ID       uuid.UUID
	Type     string
	Prompt   string
	Options  []string
	AudioURL string
}

// State is the client-facing placement state.
type State struct {
	Status          string
	QuestionsServed int
	CurrentBand     placement.Band
	NextQuestion    *NextQuestion
	AssignedLevel   domain.Level
}

// Service wires placement use cases.
type Service struct {
	repo Repo
}

func New(repo Repo) *Service { return &Service{repo: repo} }

// Current returns the active session with its next question, or
// domain.ErrNotFound when the user has no active session.
func (s *Service) Current(ctx context.Context, userID uuid.UUID) (State, error) {
	sess, err := s.repo.GetActiveSession(ctx, userID)
	if err != nil {
		return State{}, err
	}
	return s.stateWithNext(ctx, sess)
}

// Start begins a fresh session, abandoning any active one (explicit restart).
func (s *Service) Start(ctx context.Context, userID uuid.UUID) (State, error) {
	if err := s.repo.AbandonActiveSessions(ctx, userID); err != nil {
		return State{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return State{}, err
	}
	sess, err := s.repo.CreateSession(ctx, id, userID)
	if err != nil {
		return State{}, err
	}
	return s.stateWithNext(ctx, sess)
}

// SubmitAnswer scores one answer server-side, advances the band walk at
// testlet boundaries and completes the session at the hard stop.
func (s *Service) SubmitAnswer(ctx context.Context, userID, questionID uuid.UUID, answer string) (State, error) {
	sess, err := s.repo.GetActiveSession(ctx, userID)
	if err != nil {
		return State{}, fmt.Errorf("%w: no active placement session", domain.ErrConflict)
	}
	if placement.Done(sess.QuestionsServed) {
		return State{}, fmt.Errorf("%w: session already served %d questions", domain.ErrConflict, sess.QuestionsServed)
	}

	q, err := s.repo.GetQuestion(ctx, questionID)
	if err != nil {
		return State{}, err
	}
	correct := q.Correct == answer

	answerID, err := uuid.NewV7()
	if err != nil {
		return State{}, err
	}
	testlet := placement.TestletIndex(sess.QuestionsServed)
	if err := s.repo.InsertAnswer(ctx, answerID, sess.ID, questionID, testlet, correct); err != nil {
		return State{}, err // ErrConflict on repeat answer
	}
	sess.QuestionsServed++

	if placement.TestletComplete(sess.QuestionsServed) {
		answers, err := s.repo.ListAnswers(ctx, sess.ID)
		if err != nil {
			return State{}, err
		}
		bands := replayBands(answers)

		if placement.Done(sess.QuestionsServed) {
			level := placement.FinalLevel(bands[placement.MaxTestlets-2], bands[placement.MaxTestlets-1])
			if err := s.repo.CompleteAndAssignLevel(ctx, sess.ID, userID, level); err != nil {
				return State{}, err
			}
			return State{
				Status:          "completed",
				QuestionsServed: sess.QuestionsServed,
				CurrentBand:     bands[len(bands)-1],
				AssignedLevel:   level,
			}, nil
		}

		sess.CurrentBand = bands[len(bands)-1]
	}

	if err := s.repo.UpdateProgress(ctx, sess.ID, sess.CurrentBand, sess.QuestionsServed); err != nil {
		return State{}, err
	}
	return s.stateWithNext(ctx, sess)
}

// replayBands reconstructs the band each testlet was (or will be) taken at
// from the scored answers: bands[i] = band of testlet i, plus the band that
// follows the last complete testlet.
func replayBands(answers []Answer) []placement.Band {
	correctPerTestlet := make(map[int]int)
	seenPerTestlet := make(map[int]int)
	maxTestlet := -1
	for _, a := range answers {
		seenPerTestlet[a.TestletIndex]++
		if a.IsCorrect {
			correctPerTestlet[a.TestletIndex]++
		}
		if a.TestletIndex > maxTestlet {
			maxTestlet = a.TestletIndex
		}
	}

	bands := []placement.Band{placement.StartBand}
	for t := 0; t <= maxTestlet; t++ {
		if seenPerTestlet[t] < placement.TestletSize {
			break // incomplete testlet — band unchanged
		}
		bands = append(bands, placement.NextBand(bands[t], correctPerTestlet[t]))
	}
	return bands
}

func (s *Service) stateWithNext(ctx context.Context, sess Session) (State, error) {
	st := State{
		Status:          sess.Status,
		QuestionsServed: sess.QuestionsServed,
		CurrentBand:     sess.CurrentBand,
		AssignedLevel:   sess.AssignedLevel,
	}
	if sess.Status != "active" {
		return st, nil
	}
	st.Status = "active"

	q, err := s.repo.PickUnservedQuestion(ctx, sess.CurrentBand, sess.ID)
	if err != nil {
		return State{}, fmt.Errorf("pick question for band %s: %w", sess.CurrentBand, err)
	}
	st.NextQuestion = &NextQuestion{
		ID:       q.ID,
		Type:     q.Type,
		Prompt:   q.Prompt,
		Options:  q.Options,
		AudioURL: q.AudioURL,
	}
	return st, nil
}
