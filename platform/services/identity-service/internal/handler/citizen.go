package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type CitizenHandler struct {
	svc service.CitizenService
}

func NewCitizenHandler(svc service.CitizenService) *CitizenHandler {
	return &CitizenHandler{svc: svc}
}

// #region Register
func (h *CitizenHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload model.CitizenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpserver.SendError(w, r, apperr.ErrInvalidJSON)
		return
	}

	citizen, err := h.svc.RegisterCitizen(r.Context(), payload)
	if err != nil {
		httpserver.SendError(w, r, err)
		return
	}

	httpserver.JSON(w, http.StatusCreated, citizen)
}

// #region GetByID
func (h *CitizenHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("user_id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		httpserver.SendError(w, r, apperr.ErrInvalidParams.WithMeta("detail", "Niepoprawny format identyfikatora UUID"))
		return
	}

	citizen, err := h.svc.GetCitizenByID(r.Context(), userID)
	if err != nil {
		httpserver.SendError(w, r, err)
		return
	}

	httpserver.JSON(w, http.StatusOK, citizen)
}
