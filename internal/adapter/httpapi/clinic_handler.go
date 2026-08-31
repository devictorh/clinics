package httpapi

import (
	"context"
	"net/http"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

// ClinicService é o contrato de casos de uso que o adapter HTTP consome —
// interface definida no consumidor, satisfeita por service.ClinicService.
type ClinicService interface {
	Create(ctx context.Context, in service.CreateClinicInput) (domain.Clinic, error)
	Get(ctx context.Context, id string) (domain.Clinic, error)
	List(ctx context.Context) ([]domain.Clinic, error)
	Update(ctx context.Context, id string, in service.UpdateClinicInput) (domain.Clinic, error)
	UpdateBankAccount(ctx context.Context, id string, in service.BankAccountInput) (domain.Clinic, error)
	Delete(ctx context.Context, id string) error
}

var _ ClinicService = (*service.ClinicService)(nil)

type clinicHandler struct {
	svc ClinicService
}

func (h *clinicHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createClinicRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	clinic, err := h.svc.Create(r.Context(), req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toClinicResponse(clinic))
}

func (h *clinicHandler) list(w http.ResponseWriter, r *http.Request) {
	clinics, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicResponses(clinics))
}

func (h *clinicHandler) get(w http.ResponseWriter, r *http.Request) {
	clinic, err := h.svc.Get(r.Context(), r.PathValue("clinicID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicResponse(clinic))
}

func (h *clinicHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateClinicRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	clinic, err := h.svc.Update(r.Context(), r.PathValue("clinicID"), service.UpdateClinicInput{
		LegalName: req.LegalName,
		TradeName: req.TradeName,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicResponse(clinic))
}

func (h *clinicHandler) updateBankAccount(w http.ResponseWriter, r *http.Request) {
	var req bankAccountPayload
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	clinic, err := h.svc.UpdateBankAccount(r.Context(), r.PathValue("clinicID"), service.BankAccountInput{
		Bank:    req.Bank,
		Agency:  req.Agency,
		Account: req.Account,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClinicResponse(clinic))
}

func (h *clinicHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("clinicID")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
