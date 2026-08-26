package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	reqctx "github.com/zerodayz7/platform/pkg/context"
	apperr "github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type CitizenHandler struct {
	svc service.CitizenService
}

// #region NewCitizenHandler
func NewCitizenHandler(svc service.CitizenService) *CitizenHandler {
	return &CitizenHandler{svc: svc}
}

// #region Register
func (h *CitizenHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := shared.GetLogger()
	ctx := r.Context()

	// 1. Walidacja tożsamości
	actorID := reqctx.GetUserID(ctx)
	if actorID == uuid.Nil {
		httpserver.SendError(w, r, &apperr.AppError{
			Code:    "UNAUTHORIZED",
			Type:    apperr.Unauthorized,
			Message: "Brak identyfikatora pracownika wykonującego operację.",
		})
		return
	}

	// 2. Budowanie struktury Actor z kontekstu żądania
	actor := model.Actor{
		ID:       actorID,
		Name:     "System Automatyczny",
		Role:     "SYSTEM",
		ClientIP: reqctx.GetIP(r.Context()),
	}

	if reqCtx, ok := reqctx.FromContext(ctx); ok && reqCtx != nil {
		actor.Role = reqCtx.Role

		if reqCtx.Role == "OFFICER" {
			if reqCtx.Username != "" {
				actor.Name = reqCtx.Username
			}
			actor.DepartmentID = reqCtx.DepartmentID
			actor.InstitutionID = reqCtx.InstitutionID
		}
	}

	// 3. Dekodowanie payloadu HTTP
	var payload model.CitizenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Error("Błąd dekodowania JSON w Register", "error", err)
		httpserver.SendError(w, r, apperr.ErrInvalidJSON)
		return
	}

	// 4. Wywołanie serwisu z gotowym obiektem `actor`
	citizen, err := h.svc.RegisterCitizen(ctx, actor, payload)
	if err != nil {
		httpserver.SendError(w, r, err)
		return
	}

	httpserver.JSON(w, http.StatusCreated, citizen)
}

// #region GetByID
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

// #region DownloadAgreementPDF
// #region DownloadAgreementPDF
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
