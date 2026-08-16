package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/model"
	"github.com/zerodayz7/services/officer-bff/internal/service"
)

type OfficialHandler struct {
	svc service.OfficialService
}

func NewOfficialHandler(svc service.OfficialService) *OfficialHandler {
	return &OfficialHandler{svc: svc}
}

func (h *OfficialHandler) RegisterCitizen(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterCitizenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.RegisterCitizenWorkflow(r.Context(), req)
	if err != nil {
		http.Error(w, "Failed to process registration workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
