package service

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// CreatePaymentInput são os dados de criação de uma cobrança; o dentista
// é opcional.
type CreatePaymentInput struct {
	ClinicID    string
	AmountCents int64
	DentistID   string
}

// PaymentService implementa os casos de uso de cobranças Pix. A
// confirmação do pagamento é simulada em background: cada cobrança criada
// agenda um worker que, após o delay configurado, transita o status para
// approved — o equivalente ao webhook de um provedor real.
type PaymentService struct {
	payments port.PaymentRepository
	clinics  port.ClinicRepository
	dentists port.DentistRepository
	pix      port.PixProvider
	delay    func() time.Duration

	stopOnce sync.Once
	stop     chan struct{}
	wg       sync.WaitGroup
}

// PaymentOption configura o PaymentService.
type PaymentOption func(*PaymentService)

// WithApprovalDelay substitui o delay de confirmação — os testes usam
// zero para aprovação determinística.
func WithApprovalDelay(delay func() time.Duration) PaymentOption {
	return func(s *PaymentService) { s.delay = delay }
}

// NewPaymentService cria o service de cobranças. Por padrão a confirmação
// simulada acontece entre 2 e 5 segundos após a criação.
func NewPaymentService(
	payments port.PaymentRepository,
	clinics port.ClinicRepository,
	dentists port.DentistRepository,
	pix port.PixProvider,
	opts ...PaymentOption,
) *PaymentService {
	s := &PaymentService{
		payments: payments,
		clinics:  clinics,
		dentists: dentists,
		pix:      pix,
		delay: func() time.Duration {
			return 2*time.Second + rand.N(3*time.Second)
		},
		stop: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create valida os vínculos, gera a cobrança no provedor e agenda a
// confirmação simulada. Cobranças nunca são criadas para clínica ou
// dentista excluídos.
func (s *PaymentService) Create(ctx context.Context, in CreatePaymentInput) (domain.Payment, error) {
	amount, err := domain.NewAmount(in.AmountCents)
	if err != nil {
		return domain.Payment{}, err
	}
	clinic, err := s.clinics.Get(ctx, in.ClinicID)
	if err != nil {
		return domain.Payment{}, err
	}
	if in.DentistID != "" {
		dentist, err := s.dentists.Get(ctx, in.DentistID)
		if err != nil {
			return domain.Payment{}, err
		}
		if dentist.ClinicID != clinic.ID {
			return domain.Payment{}, domain.ErrNotFound
		}
	}

	pixCode, err := s.pix.GenerateCharge(ctx, port.PixChargeInput{
		Amount:       amount,
		MerchantName: clinic.TradeName,
	})
	if err != nil {
		return domain.Payment{}, err
	}
	payment, err := domain.NewPayment(clinic.ID, in.DentistID, amount, pixCode)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := s.payments.Create(ctx, *payment); err != nil {
		return domain.Payment{}, err
	}

	s.scheduleApproval(payment.ID)
	return *payment, nil
}

// Get retorna uma cobrança.
func (s *PaymentService) Get(ctx context.Context, id string) (domain.Payment, error) {
	return s.payments.Get(ctx, id)
}

// ListByClinic retorna o histórico de cobranças de uma clínica ativa.
func (s *PaymentService) ListByClinic(ctx context.Context, clinicID string) ([]domain.Payment, error) {
	if _, err := s.clinics.Get(ctx, clinicID); err != nil {
		return nil, err
	}
	return s.payments.ListByClinic(ctx, clinicID)
}

// Shutdown interrompe as confirmações pendentes e aguarda os workers
// encerrarem, respeitando o prazo do contexto.
func (s *PaymentService) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() { close(s.stop) })

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *PaymentService) scheduleApproval(paymentID string) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		timer := time.NewTimer(s.delay())
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-s.stop:
			// se o delay já venceu junto com o stop, a aprovação tem
			// prioridade — mantém o shutdown determinístico
			select {
			case <-timer.C:
			default:
				return
			}
		}

		if _, err := s.payments.Approve(context.Background(), paymentID); err != nil &&
			!errors.Is(err, domain.ErrInvalidStatusTransition) {
			slog.Warn("falha na confirmação simulada do pagamento",
				slog.String("payment_id", paymentID), slog.Any("error", err))
		}
	}()
}
