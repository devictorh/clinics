package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/devictorh/clinics/internal/adapter/memory"
	"github.com/devictorh/clinics/internal/core/domain"
)

func newPayment(t *testing.T, clinicID string) domain.Payment {
	t.Helper()
	amount, err := domain.NewAmount(15000)
	if err != nil {
		t.Fatalf("NewAmount: %v", err)
	}
	p, err := domain.NewPayment(clinicID, "", amount, "00020126...simulado")
	if err != nil {
		t.Fatalf("NewPayment: %v", err)
	}
	return *p
}

func TestPaymentRepoCreateGet(t *testing.T) {
	t.Parallel()
	repo := memory.NewPaymentRepository()
	ctx := context.Background()

	payment := newPayment(t, "clinic-A")
	if err := repo.Create(ctx, payment); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != payment.ID || got.Status != domain.PaymentStatusPending {
		t.Errorf("Get = %+v", got)
	}

	if _, err := repo.Get(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestPaymentRepoListByClinic(t *testing.T) {
	t.Parallel()
	repo := memory.NewPaymentRepository()
	ctx := context.Background()
	base := time.Now().UTC()

	segundo := newPayment(t, "clinic-A")
	segundo.CreatedAt = at(base, 2*time.Second)
	primeiro := newPayment(t, "clinic-A")
	primeiro.CreatedAt = at(base, 1*time.Second)
	outra := newPayment(t, "clinic-B")

	for _, p := range []domain.Payment{segundo, primeiro, outra} {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := repo.ListByClinic(ctx, "clinic-A")
	if err != nil {
		t.Fatalf("ListByClinic: %v", err)
	}
	if len(list) != 2 || list[0].ID != primeiro.ID || list[1].ID != segundo.ID {
		t.Errorf("ListByClinic = %d itens (esperava primeiro, segundo)", len(list))
	}

	if vazia, _ := repo.ListByClinic(ctx, "clinic-Z"); len(vazia) != 0 {
		t.Errorf("clínica sem cobranças devia listar vazio, veio %d", len(vazia))
	}
}

func TestPaymentRepoApprove(t *testing.T) {
	t.Parallel()
	repo := memory.NewPaymentRepository()
	ctx := context.Background()

	payment := newPayment(t, "clinic-A")
	if err := repo.Create(ctx, payment); err != nil {
		t.Fatalf("Create: %v", err)
	}

	approved, err := repo.Approve(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != domain.PaymentStatusApproved || approved.ApprovedAt == nil {
		t.Errorf("aprovado = %+v", approved)
	}

	stored, _ := repo.Get(ctx, payment.ID)
	if stored.Status != domain.PaymentStatusApproved {
		t.Error("aprovação não persistida")
	}

	if _, err := repo.Approve(ctx, payment.ID); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("aprovação repetida: erro = %v, quer ErrInvalidStatusTransition", err)
	}
	if _, err := repo.Approve(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("aprovação de inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestPaymentRepoConcorrencia(t *testing.T) {
	t.Parallel()
	repo := memory.NewPaymentRepository()
	ctx := context.Background()

	seed := newPayment(t, "clinic-A")
	if err := repo.Create(ctx, seed); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var wg sync.WaitGroup
	for i := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_ = repo.Create(ctx, newPayment(t, "clinic-A"))
			case 1:
				_, _ = repo.ListByClinic(ctx, "clinic-A")
			default:
				_, _ = repo.Approve(ctx, seed.ID)
			}
		}()
	}
	wg.Wait()

	list, _ := repo.ListByClinic(ctx, "clinic-A")
	if len(list) != 11 {
		t.Errorf("histórico = %d cobranças, quer 11", len(list))
	}
}
