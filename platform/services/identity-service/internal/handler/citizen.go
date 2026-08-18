package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type CitizenHandler struct {
	svc service.CitizenService
}

func NewCitizenHandler(svc service.CitizenService) *CitizenHandler {
	return &CitizenHandler{svc: svc}
}

func (h *CitizenHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload model.CitizenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	citizen, err := h.svc.RegisterCitizen(r.Context(), payload)
	if err != nil {
		http.Error(w, "Failed to register citizen: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(citizen)
}

func (h *CitizenHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("user_id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user_id UUID", http.StatusBadRequest)
		return
	}

	citizen, err := h.svc.GetCitizenByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Citizen not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(citizen)
}
