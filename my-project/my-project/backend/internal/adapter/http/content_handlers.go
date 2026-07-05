package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
)

// contentHandlers serves /tracks, /lessons and written/listening attempts.
type contentHandlers struct {
	svc *lessons.Service
}

type lessonSummaryDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	XPReward  int    `json:"xpReward"`
	Completed bool   `json:"completed"`
}

type moduleDTO struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	ThemeType       string             `json:"themeType"`
	DifficultyLevel string             `json:"difficultyLevel"`
	Locked          bool               `json:"locked"`
	Lessons         []lessonSummaryDTO `json:"lessons"`
}

type exerciseDTO struct {
	ID             string   `json:"id"`
	ExerciseType   string   `json:"exerciseType"`
	PromptContext  string   `json:"promptContext"`
	Options        []string `json:"options"`
	AudioAssetURL  *string  `json:"audioAssetUrl"`
	TargetSentence *string  `json:"targetSentence"`
}

type lessonDTO struct {
	ID               string        `json:"id"`
	Title            string        `json:"title"`
	PedagogicalFocus string        `json:"pedagogicalFocus"`
	XPReward         int           `json:"xpReward"`
	Exercises        []exerciseDTO `json:"exercises"`
}

func toExerciseDTO(e lessons.ExerciseDTO) exerciseDTO {
	dto := exerciseDTO{
		ID:            e.ID.String(),
		ExerciseType:  string(e.Type),
		PromptContext: e.Prompt,
		Options:       e.Options,
	}
	if e.AudioURL != "" {
		u := e.AudioURL
		dto.AudioAssetURL = &u
	}
	if e.TargetSentence != "" {
		t := e.TargetSentence
		dto.TargetSentence = &t
	}
	return dto
}

// GET /tracks
func (h *contentHandlers) tracks(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	modules, err := h.svc.Tracks(r.Context(), u.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]moduleDTO, 0, len(modules))
	for _, m := range modules {
		lessonsOut := make([]lessonSummaryDTO, 0, len(m.Lessons))
		for _, l := range m.Lessons {
			lessonsOut = append(lessonsOut, lessonSummaryDTO{
				ID: l.ID.String(), Title: l.Title, XPReward: l.XPReward, Completed: l.Completed,
			})
		}
		out = append(out, moduleDTO{
			ID:              m.ID.String(),
			Title:           m.Title,
			Description:     m.Descr,
			ThemeType:       m.Theme,
			DifficultyLevel: string(m.Difficulty),
			Locked:          m.Locked,
			Lessons:         lessonsOut,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /lessons/{lessonId}
func (h *contentHandlers) lesson(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	lessonID, err := uuid.Parse(chi.URLParam(r, "lessonId"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "lessonId deve ser um UUID")
		return
	}
	l, err := h.svc.Lesson(r.Context(), u.ID, lessonID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	dto := lessonDTO{
		ID:               l.ID.String(),
		Title:            l.Title,
		PedagogicalFocus: l.Focus,
		XPReward:         l.XPReward,
		Exercises:        make([]exerciseDTO, 0, len(l.Exercises)),
	}
	for _, e := range l.Exercises {
		dto.Exercises = append(dto.Exercises, toExerciseDTO(e))
	}
	writeJSON(w, http.StatusOK, dto)
}

// attemptResultDTO is the contract AttemptResult schema.
type attemptResultDTO struct {
	Correct         bool     `json:"correct"`
	AccuracyScore   float64  `json:"accuracyScore"`
	ToleratedTypos  []string `json:"toleratedTypos"`
	ExpectedAnswer  *string  `json:"expectedAnswer"`
	LessonCompleted bool     `json:"lessonCompleted"`
	XPAwarded       int      `json:"xpAwarded"`
}

// POST /exercises/{exerciseId}/attempts
func (h *contentHandlers) attempt(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	exerciseID, err := uuid.Parse(chi.URLParam(r, "exerciseId"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "exerciseId deve ser um UUID")
		return
	}
	var in struct {
		AttemptID   string  `json:"attemptId"`
		Answer      string  `json:"answer"`
		CompletedAt *string `json:"completedAt"`
		IsReview    bool    `json:"isReview"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	attemptID, err := uuid.Parse(in.AttemptID)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "attemptId deve ser um UUID")
		return
	}
	if len(in.Answer) > 1000 {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "resposta longa demais")
		return
	}
	var completedAt time.Time
	if in.CompletedAt != nil {
		completedAt, err = time.Parse(time.RFC3339, *in.CompletedAt)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "Requisição inválida", "completedAt deve ser RFC 3339")
			return
		}
	}

	res, err := h.svc.SubmitAttempt(r.Context(), u.ID, exerciseID, attemptID, in.Answer, completedAt, in.IsReview)
	if err != nil {
		writeError(w, r, err)
		return
	}

	dto := attemptResultDTO{
		Correct:         res.Correct,
		AccuracyScore:   res.AccuracyScore,
		ToleratedTypos:  orEmpty(res.ToleratedTypos),
		LessonCompleted: res.LessonCompleted,
		XPAwarded:       res.XPAwarded,
	}
	if res.ExpectedAnswer != "" {
		e := res.ExpectedAnswer
		dto.ExpectedAnswer = &e
	}
	status := http.StatusCreated
	if res.Duplicate {
		status = http.StatusOK // replayed outbox row
	}
	writeJSON(w, status, dto)
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
