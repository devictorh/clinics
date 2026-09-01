package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

func newDentistService() (*service.DentistService, *fakeClinicRepo, *fakeDentistRepo) {
	clinics := newFakeClinicRepo()
	dentists := newFakeDentistRepo()
	return service.NewDentistService(dentists, clinics), clinics, dentists
}

func validDentistInput() service.DentistInput {
	return service.DentistInput{
		Name: "Dra. Ana Souza", Phone: "(11) 98765-4321", Email: "ana@x.com", Admin: true,
	}
}

func TestDentistServiceCreate(t *testing.T) {
	t.Parallel()
	svc, clinics, dentists := newDentistService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)

	got, err := svc.Create(ctx, clinic.ID, validDentistInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ClinicID != clinic.ID || got.Phone != "11987654321" || !got.Admin {
		t.Errorf("dentista criado = %+v", got)
	}
	if _, ok := dentists.items[got.ID]; !ok {
		t.Error("dentista não persistido")
	}
}

func TestDentistServiceCreateErros(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("clínica inexistente", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := newDentistService()
		if _, err := svc.Create(ctx, "inexistente", validDentistInput()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("clínica excluída", func(t *testing.T) {
		t.Parallel()
		svc, clinics, _ := newDentistService()
		clinic := seedClinic(t, clinics)
		deleted := clinics.items[clinic.ID]
		deleted.Delete()
		clinics.items[clinic.ID] = deleted

		if _, err := svc.Create(ctx, clinic.ID, validDentistInput()); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("erro = %v, quer ErrNotFound", err)
		}
	})

	t.Run("dados inválidos", func(t *testing.T) {
		t.Parallel()
		svc, clinics, _ := newDentistService()
		clinic := seedClinic(t, clinics)
		in := validDentistInput()
		in.Email = "sem-arroba"

		if _, err := svc.Create(ctx, clinic.ID, in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("erro = %v, quer ErrInvalidInput", err)
		}
	})

	t.Run("email duplicado propagado do repositório", func(t *testing.T) {
		t.Parallel()
		svc, clinics, dentists := newDentistService()
		clinic := seedClinic(t, clinics)
		dentists.createErr = domain.ErrEmailAlreadyExists

		if _, err := svc.Create(ctx, clinic.ID, validDentistInput()); !errors.Is(err, domain.ErrEmailAlreadyExists) {
			t.Errorf("erro = %v, quer ErrEmailAlreadyExists", err)
		}
	})
}

func TestDentistServiceGet(t *testing.T) {
	t.Parallel()
	svc, clinics, dentists := newDentistService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)
	dentist := seedDentist(t, dentists, clinic.ID)

	got, err := svc.Get(ctx, clinic.ID, dentist.ID)
	if err != nil || got.ID != dentist.ID {
		t.Errorf("Get = (%+v, %v)", got, err)
	}

	if _, err := svc.Get(ctx, clinic.ID, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("dentista inexistente: erro = %v, quer ErrNotFound", err)
	}
	if _, err := svc.Get(ctx, "outra-clinica", dentist.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("dentista de outra clínica: erro = %v, quer ErrNotFound", err)
	}
}

func TestDentistServiceListByClinic(t *testing.T) {
	t.Parallel()
	svc, clinics, dentists := newDentistService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)
	seedDentist(t, dentists, clinic.ID)
	seedDentist(t, dentists, "outra-clinica")

	list, err := svc.ListByClinic(ctx, clinic.ID)
	if err != nil || len(list) != 1 {
		t.Errorf("ListByClinic = (%d itens, %v), quer 1", len(list), err)
	}

	if _, err := svc.ListByClinic(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("clínica inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestDentistServiceUpdate(t *testing.T) {
	t.Parallel()
	svc, clinics, dentists := newDentistService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)
	dentist := seedDentist(t, dentists, clinic.ID)

	in := service.DentistInput{Name: "Dr. Bruno", Phone: "11 3265-4321", Email: "bruno@x.com", Admin: false}
	got, err := svc.Update(ctx, clinic.ID, dentist.ID, in)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "Dr. Bruno" || got.Admin || dentists.items[dentist.ID].Name != "Dr. Bruno" {
		t.Error("alteração não aplicada/persistida")
	}

	if _, err := svc.Update(ctx, "outra-clinica", dentist.ID, in); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("outra clínica: erro = %v, quer ErrNotFound", err)
	}

	in.Phone = "123"
	if _, err := svc.Update(ctx, clinic.ID, dentist.ID, in); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("telefone inválido: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestDentistServiceDelete(t *testing.T) {
	t.Parallel()
	svc, clinics, dentists := newDentistService()
	ctx := context.Background()

	clinic := seedClinic(t, clinics)
	dentist := seedDentist(t, dentists, clinic.ID)

	if err := svc.Delete(ctx, clinic.ID, dentist.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if stored := dentists.items[dentist.ID]; !stored.IsDeleted() {
		t.Error("dentista não foi excluído")
	}

	if err := svc.Delete(ctx, clinic.ID, dentist.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("delete repetido: erro = %v, quer ErrNotFound", err)
	}
	if err := svc.Delete(ctx, "outra-clinica", dentist.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("outra clínica: erro = %v, quer ErrNotFound", err)
	}
}
