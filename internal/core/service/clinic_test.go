package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

func newClinicService() (*service.ClinicService, *fakeClinicRepo, *fakeDentistRepo) {
	clinics := newFakeClinicRepo()
	dentists := newFakeDentistRepo()
	return service.NewClinicService(clinics, dentists), clinics, dentists
}

func TestClinicServiceCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("sem dados bancários", func(t *testing.T) {
		t.Parallel()
		svc, clinics, _ := newClinicService()
		got, err := svc.Create(ctx, service.CreateClinicInput{
			Document:  "11.222.333/0001-81",
			LegalName: "Clínica Sorriso LTDA",
			TradeName: "Sorriso Odonto",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if got.ID == "" || got.Document.String() != "11222333000181" {
			t.Errorf("clínica criada = %+v", got)
		}
		if _, ok := clinics.items[got.ID]; !ok {
			t.Error("clínica não persistida")
		}
	})

	t.Run("com dados bancários", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newClinicService()
		got, err := svc.Create(ctx, service.CreateClinicInput{
			Document:  "529.982.247-25",
			LegalName: "Consultório Dra. Ana",
			TradeName: "Ana Odonto",
			BankAccount: &service.BankAccountInput{
				Bank: "341", Agency: "0001", Account: "12345-6",
			},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if got.BankAccount.IsZero() || got.BankAccount.Bank != "341" {
			t.Errorf("dados bancários = %+v", got.BankAccount)
		}
	})
}

func TestClinicServiceCreateErros(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	valido := service.CreateClinicInput{
		Document: "11222333000181", LegalName: "Razão LTDA", TradeName: "Fantasia",
	}

	t.Run("documento inválido", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newClinicService()
		in := valido
		in.Document = "123"
		if _, err := svc.Create(ctx, in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("erro = %v, quer ErrInvalidInput", err)
		}
	})

	t.Run("razão social vazia", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newClinicService()
		in := valido
		in.LegalName = " "
		if _, err := svc.Create(ctx, in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("erro = %v, quer ErrInvalidInput", err)
		}
	})

	t.Run("dados bancários incompletos", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newClinicService()
		in := valido
		in.BankAccount = &service.BankAccountInput{Bank: "341"}
		if _, err := svc.Create(ctx, in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("erro = %v, quer ErrInvalidInput", err)
		}
	})

	t.Run("documento duplicado propagado do repositório", func(t *testing.T) {
		t.Parallel()
		svc, clinics, _ := newClinicService()
		clinics.createErr = domain.ErrDocumentAlreadyExists
		if _, err := svc.Create(ctx, valido); !errors.Is(err, domain.ErrDocumentAlreadyExists) {
			t.Errorf("erro = %v, quer ErrDocumentAlreadyExists", err)
		}
	})
}

func TestClinicServiceGetList(t *testing.T) {
	t.Parallel()
	svc, clinics, _ := newClinicService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)

	got, err := svc.Get(ctx, clinic.ID)
	if err != nil || got.ID != clinic.ID {
		t.Errorf("Get = (%+v, %v)", got, err)
	}
	if _, err := svc.Get(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get inexistente: erro = %v, quer ErrNotFound", err)
	}

	list, err := svc.List(ctx)
	if err != nil || len(list) != 1 {
		t.Errorf("List = (%d itens, %v), quer 1", len(list), err)
	}
}

func TestClinicServiceUpdate(t *testing.T) {
	t.Parallel()
	svc, clinics, _ := newClinicService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)

	got, err := svc.Update(ctx, clinic.ID, service.UpdateClinicInput{
		LegalName: "Nova Razão LTDA", TradeName: "Novo Fantasia",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.LegalName != "Nova Razão LTDA" || clinics.items[clinic.ID].LegalName != "Nova Razão LTDA" {
		t.Error("alteração não aplicada/persistida")
	}

	if _, err := svc.Update(ctx, "inexistente", service.UpdateClinicInput{LegalName: "A", TradeName: "B"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update inexistente: erro = %v, quer ErrNotFound", err)
	}
	if _, err := svc.Update(ctx, clinic.ID, service.UpdateClinicInput{LegalName: "", TradeName: "B"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Update inválido: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestClinicServiceUpdateBankAccount(t *testing.T) {
	t.Parallel()
	svc, clinics, _ := newClinicService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)

	got, err := svc.UpdateBankAccount(ctx, clinic.ID, service.BankAccountInput{
		Bank: "341", Agency: "0001", Account: "12345-6",
	})
	if err != nil {
		t.Fatalf("UpdateBankAccount: %v", err)
	}
	if got.BankAccount.Bank != "341" {
		t.Errorf("BankAccount = %+v", got.BankAccount)
	}

	if _, err := svc.UpdateBankAccount(ctx, "inexistente", service.BankAccountInput{Bank: "341", Agency: "1", Account: "2"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("clínica inexistente: erro = %v, quer ErrNotFound", err)
	}
	if _, err := svc.UpdateBankAccount(ctx, clinic.ID, service.BankAccountInput{Bank: "341"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("dados incompletos: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestClinicServiceDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("exclui e dispara cascata", func(t *testing.T) {
		t.Parallel()
		svc, clinics, dentists := newClinicService()
		clinic := seedClinic(t, clinics)
		dentist := seedDentist(t, dentists, clinic.ID)

		if err := svc.Delete(ctx, clinic.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if len(dentists.cascadeCalls) != 1 || dentists.cascadeCalls[0] != clinic.ID {
			t.Errorf("cascata = %v, quer [%s]", dentists.cascadeCalls, clinic.ID)
		}
		if stored := dentists.items[dentist.ID]; !stored.IsDeleted() {
			t.Error("dentista não foi excluído em cascata")
		}
	})

	t.Run("inexistente não dispara cascata", func(t *testing.T) {
		t.Parallel()
		svc, _, dentists := newClinicService()

		if err := svc.Delete(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("erro = %v, quer ErrNotFound", err)
		}
		if len(dentists.cascadeCalls) != 0 {
			t.Error("cascata não deveria ter sido chamada")
		}
	})

	t.Run("falha na cascata é propagada", func(t *testing.T) {
		t.Parallel()
		svc, clinics, dentists := newClinicService()
		clinic := seedClinic(t, clinics)
		dentists.cascadeErr = errors.New("falha na cascata")

		if err := svc.Delete(ctx, clinic.ID); err == nil {
			t.Error("erro da cascata deveria ser propagado")
		}
	})
}
