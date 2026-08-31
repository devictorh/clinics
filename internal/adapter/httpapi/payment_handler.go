package httpapi

import (
	"context"
	"net/http"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

// PaymentService é o contrato de casos de uso de cobranças consumido pelo
// adapter HTTP, satisfeito por service.PaymentService.
type PaymentService interface {
	Create(ctx context.Context, in service.CreatePaymentInput) (domain.Payment, error)
	Get(ctx context.Context, id string) (domain.Payment, error)
	ListByClinic(ctx context.Context, clinicID string) ([]domain.Payment, error)
}

var _ PaymentService = (*service.PaymentService)(nil)

type paymentHandler struct {
	svc PaymentService
}

func (h *paymentHandler) create(w http.ResponseWriter, r *http.Request) {
	var req paymentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	payment, err := h.svc.Create(r.Context(), service.CreatePaymentInput{
		ClinicID:    req.ClinicID,
		AmountCents: req.Amount,
		DentistID:   req.DentistID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPaymentResponse(payment))
}

func (h *paymentHandler) get(w http.ResponseWriter, r *http.Request) {
	payment, err := h.svc.Get(r.Context(), r.PathValue("paymentID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPaymentResponse(payment))
}

func (h *paymentHandler) listByClinic(w http.ResponseWriter, r *http.Request) {
	payments, err := h.svc.ListByClinic(r.Context(), r.PathValue("clinicID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPaymentResponses(payments))
}
