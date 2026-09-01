package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// PaymentRepository implementa port.PaymentRepository sobre map + RWMutex,
// com índice por clínica para a listagem do histórico. A transição de
// status acontece sob o write lock, e leituras devolvem cópias — o worker
// de aprovação nunca compartilha memória mutável com leitores.
type PaymentRepository struct {
	mu          sync.RWMutex
	payments    map[string]domain.Payment
	clinicIndex map[string][]string
}

var _ port.PaymentRepository = (*PaymentRepository)(nil)

// NewPaymentRepository cria um repositório de cobranças vazio.
func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		payments:    make(map[string]domain.Payment),
		clinicIndex: make(map[string][]string),
	}
}

// Create persiste a cobrança e a indexa pela clínica.
func (r *PaymentRepository) Create(_ context.Context, payment domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.payments[payment.ID]; ok {
		return fmt.Errorf("%w: id já cadastrado", domain.ErrInvalidInput)
	}
	r.payments[payment.ID] = payment
	r.clinicIndex[payment.ClinicID] = append(r.clinicIndex[payment.ClinicID], payment.ID)
	return nil
}

// Get retorna a cobrança com o ID informado.
func (r *PaymentRepository) Get(_ context.Context, id string) (domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	payment, ok := r.payments[id]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}
	return payment, nil
}

// ListByClinic retorna o histórico de cobranças da clínica ordenado por
// criação.
func (r *PaymentRepository) ListByClinic(_ context.Context, clinicID string) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.clinicIndex[clinicID]
	payments := make([]domain.Payment, 0, len(ids))
	for _, id := range ids {
		payments = append(payments, r.payments[id])
	}
	sortByCreation(payments, func(p domain.Payment) (time.Time, string) {
		return p.CreatedAt, p.ID
	})
	return payments, nil
}

// Approve transita a cobrança para approved sob o write lock e devolve o
// registro atualizado.
func (r *PaymentRepository) Approve(_ context.Context, id string) (domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	payment, ok := r.payments[id]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}
	if err := payment.Approve(); err != nil {
		return domain.Payment{}, err
	}
	r.payments[id] = payment
	return payment, nil
}
