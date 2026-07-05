package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	speechuc "github.com/pradella/fluentdev/backend/internal/usecase/speech"
)

// speechHandlers serves POST /exercises/{exerciseId}/speech-attempts.
type speechHandlers struct {
	svc *speechuc.Service
}

// speechResultDTO is the contract SpeechResult schema.
type speechResultDTO struct {
	Similarity  float64  `json:"similarity"`
	Passed      bool     `json:"passed"`
	Transcript  string   `json:"transcript"`
	MissedWords []string `json:"missedWords"`
	XPAwarded   int      `json:"xpAwarded"`
}

func (h *speechHandlers) attempt(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	exerciseID, err := uuid.Parse(chi.URLParam(r, "exerciseId"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "exerciseId deve ser um UUID")
		return
	}

	// Cap the whole multipart body slightly above the audio limit.
	r.Body = http.MaxBytesReader(w, r.Body, speechuc.MaxAudioBytes+64*1024)
	if err := r.ParseMultipartForm(speechuc.MaxAudioBytes + 64*1024); err != nil {
		writeProblem(w, http.StatusRequestEntityTooLarge, "Áudio grande demais",
			"O áudio deve ter no máximo 30 segundos (1,5 MB).")
		return
	}

	attemptID, err := uuid.Parse(r.FormValue("attemptId"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "attemptId deve ser um UUID")
		return
	}
	isReview := r.FormValue("isReview") == "true"

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "arquivo de áudio ausente")
		return
	}
	defer file.Close()
	if header.Size > speechuc.MaxAudioBytes {
		writeProblem(w, http.StatusRequestEntityTooLarge, "Áudio grande demais",
			"O áudio deve ter no máximo 30 segundos (1,5 MB).")
		return
	}

	res, err := h.svc.Submit(r.Context(), u.ID, exerciseID, attemptID, file, header.Size, isReview)
	if err != nil {
		switch {
		case errors.Is(err, speechuc.ErrUnintelligible):
			writeProblem(w, http.StatusUnprocessableEntity, "Áudio ininteligível",
				"Não conseguimos entender o áudio. Tente gravar de novo em um lugar mais silencioso.")
		case errors.Is(err, speechuc.ErrProvidersUnavailable):
			writeProblem(w, http.StatusServiceUnavailable, "Serviço de fala indisponível",
				"A avaliação de fala está temporariamente indisponível. Tente novamente em instantes.")
		default:
			writeError(w, r, err)
		}
		return
	}

	status := http.StatusCreated
	if res.Duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, speechResultDTO{
		Similarity:  res.Similarity,
		Passed:      res.Passed,
		Transcript:  res.Transcript,
		MissedWords: orEmpty(res.MissedWords),
		XPAwarded:   res.XPAwarded,
	})
}
