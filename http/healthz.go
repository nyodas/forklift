package http

import (
	"net/http"
)

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK
	w.WriteHeader(statusCode)
	w.Write(nil)
}
