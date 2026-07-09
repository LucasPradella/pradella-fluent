package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
)

// placementHandlers serves /placement/*.
type placementHandlers struct {
	svc *placementuc.Service
}

// placementStateDTO is the contract PlacementState schema.
type placementStateDTO struct {
	Status          string           `json:"status"`
	QuestionsServed int              `json:"questionsServed"`
	CurrentBand     string           `json:"currentBand,omitempty"`
	NextQuestion    *nextQuestionDTO `json:"nextQuestion"`
	AssignedLevel   *string          `json:"assignedLevel"`
}

type nextQuestionDTO struct {
	ID            string   `json:"id"`
	QuestionType  string   `json:"questionType"`
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options"`
	AudioAssetURL *string  `json:"audioAssetUrl"`
}

func toPlacementDTO(st placementuc.State) placementStateDTO {
	dto := placementStateDTO{
		Status:          st.Status,
		QuestionsServed: st.QuestionsServed,
		CurrentBand:     string(st.CurrentBand),
	}
	if st.AssignedLevel != "" {
		lvl := string(st.AssignedLevel)
		dto.AssignedLevel = &lvl
	}
	if st.NextQuestion != nil {
		q := &nextQuestionDTO{
			ID:           st.NextQuestion.ID.String(),
			QuestionType: st.NextQuestion.Type,
			Prompt:       st.NextQuestion.Prompt,
			Options:      st.NextQuestion.Options,
		}
		if st.NextQuestion.AudioURL != "" {
			u := st.NextQuestion.AudioURL
			q.AudioAssetURL = &u
		}
		dto.NextQuestion = q
	}
	return dto
}

// GET /placement/session
func (h *placementHandlers) current(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	st, err := h.svc.Current(r.Context(), u.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlacementDTO(st))
}

// POST /placement/session
func (h *placementHandlers) start(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	st, err := h.svc.Start(r.Context(), u.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlacementDTO(st))
}

// POST /placement/session/answers
func (h *placementHandlers) answer(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	var in struct {
		QuestionID string `json:"questionId"`
		Answer     string `json:"answer"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	qid, err := uuid.Parse(in.QuestionID)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "questionId deve ser um UUID")
		return
	}
	if len(in.Answer) > 500 {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "resposta longa demais")
		return
	}
	st, err := h.svc.SubmitAnswer(r.Context(), u.ID, qid, in.Answer)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlacementDTO(st))
}
