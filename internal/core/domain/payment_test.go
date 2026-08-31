package domain_test

import (
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
)

func mustAmount(t *testing.T, cents int64) domain.Amount {
	t.Helper()
	a, err := domain.NewAmount(cents)
	if err != nil {
		t.Fatalf("NewAmount(%d): %v", cents, err)
	}
	return a
}

func TestNewPayment(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("clinic-1", "dentist-1", mustAmount(t, 15000), "00020126...")
	if err != nil {
		t.Fatalf("NewPayment: %v", err)
	}
	if p.ID == "" || p.Status != domain.PaymentStatusPending || p.ApprovedAt != nil {
		t.Errorf("cobrança criada = %+v", p)
	}

	sem, err := domain.NewPayment("clinic-1", "", mustAmount(t, 100), "00020126...")
	if err != nil {
		t.Fatalf("NewPayment sem dentista: %v", err)
	}
	if sem.DentistID != "" {
		t.Errorf("DentistID = %q, quer vazio", sem.DentistID)
	}
}

func TestNewPaymentInvalido(t *testing.T) {
	t.Parallel()

	amount := mustAmount(t, 100)
	tests := []struct {
		name     string
		clinicID string
		amount   domain.Amount
		pixCode  string
	}{
		{"clínica vazia", "", amount, "codigo"},
		{"amount zero value", "clinic-1", domain.Amount(0), "codigo"},
		{"código pix vazio", "clinic-1", amount, "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewPayment(tt.clinicID, "", tt.amount, tt.pixCode)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("erro = %v, quer ErrInvalidInput", err)
			}
		})
	}
}

func TestPaymentApprove(t *testing.T) {
	t.Parallel()

	p, err := domain.NewPayment("clinic-1", "", mustAmount(t, 100), "codigo")
	if err != nil {
		t.Fatalf("NewPayment: %v", err)
	}

	if err := p.Approve(); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if p.Status != domain.PaymentStatusApproved || p.ApprovedAt == nil {
		t.Errorf("após aprovação = %+v", p)
	}

	if err := p.Approve(); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("segunda aprovação: erro = %v, quer ErrInvalidStatusTransition", err)
	}
}
