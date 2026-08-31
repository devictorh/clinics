package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

type paymentFixture struct {
	svc      *service.PaymentService
	payments *fakePaymentRepo
	clinics  *fakeClinicRepo
	dentists *fakeDentistRepo
	pix      *fakePixProvider
}

func newPaymentService(t *testing.T, opts ...service.PaymentOption) paymentFixture {
	t.Helper()
	f := paymentFixture{
		payments: newFakePaymentRepo(),
		clinics:  newFakeClinicRepo(),
		dentists: newFakeDentistRepo(),
		pix:      &fakePixProvider{code: "00020126...simulado"},
	}
	opts = append([]service.PaymentOption{
		service.WithApprovalDelay(func() time.Duration { return 0 }),
	}, opts...)
	f.svc = service.NewPaymentService(f.payments, f.clinics, f.dentists, f.pix, opts...)
	return f
}

func shutdownNow(t *testing.T, svc *service.PaymentService) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestPaymentServiceCreate(t *testing.T) {
	t.Parallel()
	f := newPaymentService(t)
	ctx := context.Background()

	clinic := seedClinic(t, f.clinics)
	dentist := seedDentist(t, f.dentists, clinic.ID)

	got, err := f.svc.Create(ctx, service.CreatePaymentInput{
		ClinicID: clinic.ID, AmountCents: 15000, DentistID: dentist.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Status != domain.PaymentStatusPending || got.PixCode != "00020126...simulado" {
		t.Errorf("cobrança criada = %+v", got)
	}
	if got.DentistID != dentist.ID || got.Amount.Cents() != 15000 {
		t.Errorf("vínculos = %+v", got)
	}
	if f.pix.calls != 1 {
		t.Errorf("provedor chamado %d vezes, quer 1", f.pix.calls)
	}

	// com delay zero, o Shutdown aguarda o worker e a aprovação é determinística
	shutdownNow(t, f.svc)
	stored, ok := f.payments.get(got.ID)
	if !ok || stored.Status != domain.PaymentStatusApproved || stored.ApprovedAt == nil {
		t.Errorf("após confirmação simulada = %+v", stored)
	}
}

func TestPaymentServiceCreateErros(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("valor inválido", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 0}); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("erro = %v, quer ErrInvalidInput", err)
		}
	})

	t.Run("clínica inexistente", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: "nao-existe", AmountCents: 100}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("clínica excluída", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		deleted := f.clinics.items[clinic.ID]
		deleted.Delete()
		f.clinics.items[clinic.ID] = deleted

		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("dentista inexistente", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100, DentistID: "nao-existe"}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("dentista de outra clínica", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		outro := seedDentist(t, f.dentists, "outra-clinica")
		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100, DentistID: outro.ID}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("dentista excluído", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		dentist := seedDentist(t, f.dentists, clinic.ID)
		deleted := f.dentists.items[dentist.ID]
		deleted.Delete()
		f.dentists.items[dentist.ID] = deleted

		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100, DentistID: dentist.ID}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("falha do provedor não persiste cobrança", func(t *testing.T) {
		t.Parallel()
		f := newPaymentService(t)
		clinic := seedClinic(t, f.clinics)
		f.pix.err = errors.New("provedor indisponível")

		if _, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100}); err == nil {
			t.Fatal("erro do provedor deveria ser propagado")
		}
		if len(f.payments.items) != 0 {
			t.Error("cobrança não deveria ter sido persistida")
		}
	})
}

func TestPaymentServiceGetList(t *testing.T) {
	t.Parallel()
	f := newPaymentService(t)
	ctx := context.Background()

	clinic := seedClinic(t, f.clinics)
	created, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got, err := f.svc.Get(ctx, created.ID); err != nil || got.ID != created.ID {
		t.Errorf("Get = (%+v, %v)", got, err)
	}
	if _, err := f.svc.Get(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get inexistente: erro = %v, quer ErrNotFound", err)
	}

	if list, err := f.svc.ListByClinic(ctx, clinic.ID); err != nil || len(list) != 1 {
		t.Errorf("ListByClinic = (%d itens, %v), quer 1", len(list), err)
	}
	if _, err := f.svc.ListByClinic(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ListByClinic inexistente: erro = %v, quer ErrNotFound", err)
	}

	shutdownNow(t, f.svc)
}

func TestPaymentServiceShutdownCancelaPendentes(t *testing.T) {
	t.Parallel()
	f := newPaymentService(t, service.WithApprovalDelay(func() time.Duration { return time.Hour }))
	ctx := context.Background()

	clinic := seedClinic(t, f.clinics)
	created, err := f.svc.Create(ctx, service.CreatePaymentInput{ClinicID: clinic.ID, AmountCents: 100})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	shutdownNow(t, f.svc)

	stored, _ := f.payments.get(created.ID)
	if stored.Status != domain.PaymentStatusPending {
		t.Errorf("cobrança deveria seguir pendente após shutdown, veio %s", stored.Status)
	}
}
