package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus é o ciclo de vida de uma cobrança: pending → approved.
type PaymentStatus string

// Status possíveis de uma cobrança.
const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusApproved PaymentStatus = "approved"
)

// Payment é uma cobrança Pix recebida por uma clínica, opcionalmente
// atribuída a um dentista. É um registro imutável de histórico
// financeiro: a única mutação permitida em toda a sua vida é a transição
// de status via Approve — não há update nem exclusão, e o registro
// permanece íntegro mesmo após o soft delete da clínica.
type Payment struct {
	ID         string
	ClinicID   string
	DentistID  string
	Amount     Amount
	Status     PaymentStatus
	PixCode    string
	CreatedAt  time.Time
	ApprovedAt *time.Time
}

// NewPayment cria uma cobrança pendente para a clínica, com o código Pix
// copia-e-cola já gerado pelo provedor.
func NewPayment(clinicID, dentistID string, amount Amount, pixCode string) (*Payment, error) {
	switch {
	case strings.TrimSpace(clinicID) == "":
		return nil, invalidInput("clínica é obrigatória")
	case amount.Cents() <= 0:
		return nil, invalidInput("valor deve ser maior que zero")
	case strings.TrimSpace(pixCode) == "":
		return nil, invalidInput("código pix é obrigatório")
	}

	return &Payment{
		ID:        uuid.NewString(),
		ClinicID:  clinicID,
		DentistID: dentistID,
		Amount:    amount,
		Status:    PaymentStatusPending,
		PixCode:   pixCode,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Approve confirma o pagamento de uma cobrança pendente, registrando o
// instante da aprovação.
func (p *Payment) Approve() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidStatusTransition
	}
	now := time.Now().UTC()
	p.Status = PaymentStatusApproved
	p.ApprovedAt = &now
	return nil
}
