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

func TestDentistRepoCreateGet(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()

	dentist := newDentist(t, "clinic-A", "Dra. Ana")
	if err := repo.Create(ctx, dentist); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, dentist.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != dentist.ID || got.ClinicID != "clinic-A" || got.Name != "Dra. Ana" {
		t.Errorf("Get = %+v", got)
	}

	if _, err := repo.Get(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestDentistRepoUpdate(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()

	dentist := newDentist(t, "clinic-A", "Dra. Ana")
	if err := repo.Create(ctx, dentist); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := dentist.Update("Dra. Ana Souza", "(11) 91234-5678", "ana@x.com", true); err != nil {
		t.Fatalf("Update entidade: %v", err)
	}
	if err := repo.Update(ctx, dentist); err != nil {
		t.Fatalf("Update repo: %v", err)
	}
	got, _ := repo.Get(ctx, dentist.ID)
	if got.Name != "Dra. Ana Souza" || !got.Admin {
		t.Errorf("dados após update = %+v", got)
	}

	troca := dentist
	troca.ClinicID = "clinic-B"
	if err := repo.Update(ctx, troca); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Update trocando clínica: erro = %v, quer ErrInvalidInput", err)
	}

	if err := repo.Update(ctx, newDentist(t, "clinic-A", "Novo")); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestDentistRepoSoftDelete(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()

	dentist := newDentist(t, "clinic-A", "Dra. Ana")
	if err := repo.Create(ctx, dentist); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, dentist.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, dentist.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get de excluído: erro = %v, quer ErrNotFound", err)
	}
	if err := repo.Delete(ctx, dentist.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete repetido: erro = %v, quer ErrNotFound", err)
	}
}

func TestDentistRepoListByClinic(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()
	base := time.Now().UTC()

	segundo := newDentist(t, "clinic-A", "Segundo")
	segundo.CreatedAt = at(base, 2*time.Second)
	primeiro := newDentist(t, "clinic-A", "Primeiro")
	primeiro.CreatedAt = at(base, 1*time.Second)
	excluido := newDentist(t, "clinic-A", "Excluído")
	outraClinica := newDentist(t, "clinic-B", "Outra")

	for _, d := range []domain.Dentist{segundo, primeiro, excluido, outraClinica} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	if err := repo.Delete(ctx, excluido.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, err := repo.ListByClinic(ctx, "clinic-A")
	if err != nil {
		t.Fatalf("ListByClinic: %v", err)
	}
	if len(list) != 2 || list[0].ID != primeiro.ID || list[1].ID != segundo.ID {
		t.Errorf("ListByClinic = %d itens (esperava primeiro, segundo)", len(list))
	}

	if vazia, _ := repo.ListByClinic(ctx, "clinic-Z"); len(vazia) != 0 {
		t.Errorf("clínica sem dentistas devia listar vazio, veio %d", len(vazia))
	}
}

func TestDentistRepoDeleteByClinicID(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()

	alvo1 := newDentist(t, "clinic-A", "Alvo 1")
	alvo2 := newDentist(t, "clinic-A", "Alvo 2")
	preservado := newDentist(t, "clinic-B", "Preservado")
	for _, d := range []domain.Dentist{alvo1, alvo2, preservado} {
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	if err := repo.DeleteByClinicID(ctx, "clinic-A"); err != nil {
		t.Fatalf("DeleteByClinicID: %v", err)
	}

	if list, _ := repo.ListByClinic(ctx, "clinic-A"); len(list) != 0 {
		t.Errorf("clinic-A ainda tem %d dentistas ativos", len(list))
	}
	if _, err := repo.Get(ctx, preservado.ID); err != nil {
		t.Errorf("dentista de outra clínica foi afetado: %v", err)
	}

	if err := repo.DeleteByClinicID(ctx, "clinic-Z"); err != nil {
		t.Errorf("cascata em clínica sem dentistas devia ser nil, veio %v", err)
	}
}

func TestDentistRepoConcorrencia(t *testing.T) {
	t.Parallel()
	repo := memory.NewDentistRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clinicID := "clinic-A"
			if i%2 == 0 {
				clinicID = "clinic-B"
			}
			switch i % 3 {
			case 0:
				_ = repo.Create(ctx, newDentist(t, clinicID, "Concorrente"))
			case 1:
				_, _ = repo.ListByClinic(ctx, clinicID)
			default:
				_ = repo.DeleteByClinicID(ctx, clinicID)
			}
		}()
	}
	wg.Wait()
}
