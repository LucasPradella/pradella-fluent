package httpapi

import (
	"net/http"
	"time"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
)

// dashboardHandlers serves /dashboard and /review-queue.
type dashboardHandlers struct {
	svc *dashboard.Service
}

type heatmapDayDTO struct {
	Date         string `json:"date"`
	Interactions int    `json:"interactions"`
	Level        int    `json:"level"`
}

type dashboardDTO struct {
	CurrentStreak int             `json:"currentStreak"`
	LongestStreak int             `json:"longestStreak"`
	TotalXP       int             `json:"totalXp"`
	Heatmap       []heatmapDayDTO `json:"heatmap"`
	DueReviews    int             `json:"dueReviews"`
}

// GET /dashboard
func (h *dashboardHandlers) dashboard(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	data, err := h.svc.Dashboard(r.Context(), u.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	days := make([]heatmapDayDTO, 0, len(data.Heatmap))
	for _, d := range data.Heatmap {
		days = append(days, heatmapDayDTO{Date: d.Date, Interactions: d.Interactions, Level: d.Level})
	}
	writeJSON(w, http.StatusOK, dashboardDTO{
		CurrentStreak: data.CurrentStreak,
		LongestStreak: data.LongestStreak,
		TotalXP:       data.TotalXP,
		Heatmap:       days,
		DueReviews:    data.DueReviews,
	})
}

type reviewItemDTO struct {
	ID           string      `json:"id"`
	Exercise     exerciseDTO `json:"exercise"`
	DueAt        string      `json:"dueAt"`
	FailureCount int         `json:"failureCount"`
}

// GET /review-queue
func (h *dashboardHandlers) reviewQueue(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	items, err := h.svc.ReviewQueue(r.Context(), u.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]reviewItemDTO, 0, len(items))
	for _, it := range items {
		out = append(out, reviewItemDTO{
			ID:           it.ID.String(),
			Exercise:     toExerciseDTO(it.Exercise),
			DueAt:        it.DueAt.UTC().Format(time.RFC3339),
			FailureCount: it.FailureCount,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
