package handler

import (
	"net/http"
	"strconv"

	"github.com/thriftllm/backend/internal/store"
)

type DashboardHandler struct {
	DB *store.Postgres
}

func (h *DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetOverviewStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get overview stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *DashboardHandler) Usage(w http.ResponseWriter, r *http.Request) {
	rangeStr := r.URL.Query().Get("range")
	days := 7
	switch rangeStr {
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "7d", "":
		days = 7
	}

	data, err := h.DB.GetUsageOverTime(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get usage data")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *DashboardHandler) ModelBreakdown(w http.ResponseWriter, r *http.Request) {
	rangeStr := r.URL.Query().Get("range")
	days := 7
	switch rangeStr {
	case "30d":
		days = 30
	case "90d":
		days = 90
	}

	data, err := h.DB.GetModelUsageBreakdown(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get model breakdown")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *DashboardHandler) Requests(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := store.RequestLogFilter{
		Page:     page,
		Limit:    limit,
		Search:   r.URL.Query().Get("search"),
		Model:    r.URL.Query().Get("model"),
		Provider: r.URL.Query().Get("provider"),
	}

	if c := r.URL.Query().Get("cache"); c != "" {
		val := c == "true"
		filter.CacheHit = &val
	}
	if s := r.URL.Query().Get("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			filter.Status = &v
		}
	}

	result, err := h.DB.ListRequestLogs(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
