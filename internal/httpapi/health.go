package httpapi

import "net/http"

type healthResponse struct {
	Status string `json:"status"` // "ok" | "degraded"
	DB     string `json:"db"`     // "ok" | "error"
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{Status: "ok", DB: "ok"}
	code := http.StatusOK
	if err := h.deps.DB.PingContext(r.Context()); err != nil {
		h.deps.Logger.Error("health: db ping failed", "err", err)
		resp = healthResponse{Status: "degraded", DB: "error"}
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, resp)
}
