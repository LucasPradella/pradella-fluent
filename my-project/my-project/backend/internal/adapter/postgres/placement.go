package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/placement"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
)

// PlacementRepo implements placementuc.Repo.
type PlacementRepo struct{ *Store }

func NewPlacementRepo(s *Store) *PlacementRepo { return &PlacementRepo{s} }

func toUCSession(s gen.PlacementSession) placementuc.Session {
	out := placementuc.Session{
		ID:              s.ID,
		UserID:          s.UserID,
		Status:          string(s.Status),
		CurrentBand:     placement.Band(s.CurrentBand),
		QuestionsServed: int(s.QuestionsServed),
	}
	if s.AssignedLevel.Valid {
		out.AssignedLevel = domain.Level(s.AssignedLevel.ProficiencyLevel)
	}
	return out
}

func toUCQuestion(q gen.PlacementQuestion) placementuc.Question {
	return placementuc.Question{
		ID:       q.ID,
		Band:     placement.Band(q.CefrBand),
		Type:     string(q.QuestionType),
		Prompt:   q.Prompt,
		Options:  decodeOptions(q.Options),
		Correct:  q.CorrectOption,
		AudioURL: q.AudioAssetUrl.String,
	}
}

func (r *PlacementRepo) GetActiveSession(ctx context.Context, userID uuid.UUID) (placementuc.Session, error) {
	s, err := r.q.GetActivePlacementSession(ctx, userID)
	if err != nil {
		return placementuc.Session{}, mapErr(err)
	}
	return toUCSession(s), nil
}

func (r *PlacementRepo) CreateSession(ctx context.Context, id, userID uuid.UUID) (placementuc.Session, error) {
	s, err := r.q.CreatePlacementSession(ctx, gen.CreatePlacementSessionParams{ID: id, UserID: userID})
	if err != nil {
		return placementuc.Session{}, mapErr(err)
	}
	return toUCSession(s), nil
}

func (r *PlacementRepo) AbandonActiveSessions(ctx context.Context, userID uuid.UUID) error {
	return mapErr(r.q.AbandonActivePlacementSessions(ctx, userID))
}

func (r *PlacementRepo) UpdateProgress(ctx context.Context, sessionID uuid.UUID, band placement.Band, served int) error {
	return mapErr(r.q.UpdatePlacementProgress(ctx, gen.UpdatePlacementProgressParams{
		ID:              sessionID,
		CurrentBand:     gen.CefrBand(band),
		QuestionsServed: int32(served), //nolint:gosec // bounded 0..12
	}))
}

// CompleteAndAssignLevel finishes the session and sets the user's level in
// one transaction (atomic assignment — US1).
func (r *PlacementRepo) CompleteAndAssignLevel(ctx context.Context, sessionID, userID uuid.UUID, level domain.Level) error {
	return r.inTx(ctx, func(q *gen.Queries) error {
		if err := q.CompletePlacementSession(ctx, gen.CompletePlacementSessionParams{
			ID: sessionID,
			AssignedLevel: gen.NullProficiencyLevel{
				ProficiencyLevel: gen.ProficiencyLevel(level),
				Valid:            true,
			},
		}); err != nil {
			return mapErr(err)
		}
		return mapErr(q.SetProficiencyLevel(ctx, gen.SetProficiencyLevelParams{
			ID: userID,
			ProficiencyLevel: gen.NullProficiencyLevel{
				ProficiencyLevel: gen.ProficiencyLevel(level),
				Valid:            true,
			},
		}))
	})
}

func (r *PlacementRepo) InsertAnswer(ctx context.Context, id, sessionID, questionID uuid.UUID, testlet int, correct bool) error {
	return mapErr(r.q.InsertPlacementAnswer(ctx, gen.InsertPlacementAnswerParams{
		ID:                 id,
		PlacementSessionID: sessionID,
		QuestionID:         questionID,
		TestletIndex:       int32(testlet), //nolint:gosec // bounded 0..3
		IsCorrect:          correct,
	}))
}

func (r *PlacementRepo) ListAnswers(ctx context.Context, sessionID uuid.UUID) ([]placementuc.Answer, error) {
	rows, err := r.q.ListPlacementAnswers(ctx, sessionID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]placementuc.Answer, 0, len(rows))
	for _, a := range rows {
		out = append(out, placementuc.Answer{
			QuestionID:   a.QuestionID,
			TestletIndex: int(a.TestletIndex),
			IsCorrect:    a.IsCorrect,
			AnsweredAt:   a.AnsweredAt.Time,
		})
	}
	return out, nil
}

func (r *PlacementRepo) GetQuestion(ctx context.Context, id uuid.UUID) (placementuc.Question, error) {
	q, err := r.q.GetPlacementQuestion(ctx, id)
	if err != nil {
		return placementuc.Question{}, mapErr(err)
	}
	return toUCQuestion(q), nil
}

func (r *PlacementRepo) PickUnservedQuestion(ctx context.Context, band placement.Band, sessionID uuid.UUID) (placementuc.Question, error) {
	q, err := r.q.PickUnservedQuestionForBand(ctx, gen.PickUnservedQuestionForBandParams{
		CefrBand:           gen.CefrBand(band),
		PlacementSessionID: sessionID,
	})
	if err != nil {
		return placementuc.Question{}, mapErr(err)
	}
	return toUCQuestion(q), nil
}
