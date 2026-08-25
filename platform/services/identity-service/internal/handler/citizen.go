package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type CitizenHandler struct {
	svc service.CitizenService
}

//#region NewCitizenHandler
func NewCitizenHandler(svc service.CitizenService) *CitizenHandler {
	return &CitizenHandler{svc: svc}
}

// #region Register
//#region Register
func (h *CitizenHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := shared.GetLogger()
	var payload model.CitizenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Error("Błąd dekodowania JSON w Register", "error", err)
		httpserver.SendError(w, r, apperr.ErrInvalidJSON)
		return
	}

	// Diagnostyka: wypisanie zdekodowanego payloadu w konsoli
	// log.DebugJSON("Otrzymany payload w CitizenHandler.Register", payload)

	citizen, err := h.svc.RegisterCitizen(r.Context(), payload)
	if err != nil {
		httpserver.SendError(w, r, err)
		return
	}

	httpserver.JSON(w, http.StatusCreated, citizen)
}

// #region GetByID
//#region GetByID
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

// #region DownloadAgreementPDF
//#region DownloadAgreementPDF
func (h *CitizenHandler) DownloadAgreementPDF(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("agreement_id")
	agreementID, err := uuid.Parse(idStr)
	if err != nil {
		httpserver.SendError(w, r, apperr.ErrInvalidParams.WithMeta("detail", "Niepoprawny format identyfikatora UUID umowy"))
		return
	}

	pdfBytes, err := h.svc.DownloadAgreementPDF(r.Context(), agreementID)
	if err != nil {
		httpserver.SendError(w, r, err)
		return
	}

	// Ustawienie nagłówków dla pliku PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="agreement-%s.pdf"`, agreementID.String()))
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(pdfBytes)
}
