package httpapi

import (
	"context"
	"net/http"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

// DentistService é o contrato de casos de uso de dentistas consumido pelo
// adapter HTTP, satisfeito por service.DentistService.
type DentistService interface {
	Create(ctx context.Context, clinicID string, in service.DentistInput) (domain.Dentist, error)
	Get(ctx context.Context, clinicID, dentistID string) (domain.Dentist, error)
	ListByClinic(ctx context.Context, clinicID string) ([]domain.Dentist, error)
	Update(ctx context.Context, clinicID, dentistID string, in service.DentistInput) (domain.Dentist, error)
	Delete(ctx context.Context, clinicID, dentistID string) error
}

var _ DentistService = (*service.DentistService)(nil)

type dentistHandler struct {
	svc DentistService
}

func (h *dentistHandler) create(w http.ResponseWriter, r *http.Request) {
	var req dentistRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	dentist, err := h.svc.Create(r.Context(), r.PathValue("clinicID"), req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDentistResponse(dentist))
}

func (h *dentistHandler) list(w http.ResponseWriter, r *http.Request) {
	dentists, err := h.svc.ListByClinic(r.Context(), r.PathValue("clinicID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDentistResponses(dentists))
}

func (h *dentistHandler) get(w http.ResponseWriter, r *http.Request) {
	dentist, err := h.svc.Get(r.Context(), r.PathValue("clinicID"), r.PathValue("dentistID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDentistResponse(dentist))
}

func (h *dentistHandler) update(w http.ResponseWriter, r *http.Request) {
	var req dentistRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	dentist, err := h.svc.Update(r.Context(), r.PathValue("clinicID"), r.PathValue("dentistID"), req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDentistResponse(dentist))
}

func (h *dentistHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("clinicID"), r.PathValue("dentistID")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
