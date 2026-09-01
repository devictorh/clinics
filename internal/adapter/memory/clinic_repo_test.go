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

func TestClinicRepoCreateGet(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()

	clinic := newClinic(t, cnpjFor(t, 1))
	if err := repo.Create(ctx, clinic); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, clinic.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != clinic.ID || got.Document != clinic.Document || got.LegalName != clinic.LegalName {
		t.Errorf("Get = %+v, quer %+v", got, clinic)
	}

	if _, err := repo.Get(ctx, "inexistente"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get inexistente: erro = %v, quer ErrNotFound", err)
	}
}

func TestClinicRepoDocumentoDuplicado(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()

	doc := cnpjFor(t, 2)
	if err := repo.Create(ctx, newClinic(t, doc)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(ctx, newClinic(t, doc)); !errors.Is(err, domain.ErrDocumentAlreadyExists) {
		t.Errorf("Create duplicado: erro = %v, quer ErrDocumentAlreadyExists", err)
	}
}

func TestClinicRepoSoftDelete(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()

	doc := cnpjFor(t, 3)
	clinic := newClinic(t, doc)
	if err := repo.Create(ctx, clinic); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, clinic.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.Get(ctx, clinic.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get de excluída: erro = %v, quer ErrNotFound", err)
	}
	if list, _ := repo.List(ctx); len(list) != 0 {
		t.Errorf("List após delete = %d itens, quer 0", len(list))
	}
	if err := repo.Delete(ctx, clinic.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete repetido: erro = %v, quer ErrNotFound", err)
	}

	// soft delete libera o documento para novo cadastro
	if err := repo.Create(ctx, newClinic(t, doc)); err != nil {
		t.Errorf("Create com documento liberado: %v", err)
	}
}

func TestClinicRepoUpdate(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()

	clinic := newClinic(t, cnpjFor(t, 4))
	if err := repo.Create(ctx, clinic); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := clinic.Update("Nova Razão LTDA", "Novo Fantasia"); err != nil {
		t.Fatalf("Update entidade: %v", err)
	}
	if err := repo.Update(ctx, clinic); err != nil {
		t.Fatalf("Update repo: %v", err)
	}
	got, _ := repo.Get(ctx, clinic.ID)
	if got.LegalName != "Nova Razão LTDA" {
		t.Errorf("LegalName = %q após update", got.LegalName)
	}

	outra := newClinic(t, cnpjFor(t, 5))
	if err := repo.Update(ctx, outra); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update inexistente: erro = %v, quer ErrNotFound", err)
	}

	troca := clinic
	if doc, err := domain.NewDocument(cnpjFor(t, 6)); err == nil {
		troca.Document = doc
	}
	if err := repo.Update(ctx, troca); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Update trocando documento: erro = %v, quer ErrInvalidInput", err)
	}

	if err := repo.Delete(ctx, clinic.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Update(ctx, clinic); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update de excluída: erro = %v, quer ErrNotFound", err)
	}
}

func TestClinicRepoListOrdenada(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()
	base := time.Now().UTC()

	terceira := newClinic(t, cnpjFor(t, 7))
	terceira.CreatedAt = at(base, 3*time.Second)
	primeira := newClinic(t, cnpjFor(t, 8))
	primeira.CreatedAt = at(base, 1*time.Second)
	segunda := newClinic(t, cnpjFor(t, 9))
	segunda.CreatedAt = at(base, 2*time.Second)

	for _, c := range []domain.Clinic{terceira, primeira, segunda} {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{primeira.ID, segunda.ID, terceira.ID}
	if len(list) != 3 {
		t.Fatalf("List = %d itens, quer 3", len(list))
	}
	for i, c := range list {
		if c.ID != want[i] {
			t.Errorf("posição %d = %s, quer %s", i, c.ID, want[i])
		}
	}
}

func TestClinicRepoDevolveCopias(t *testing.T) {
	t.Parallel()
	repo := memory.NewClinicRepository()
	ctx := context.Background()

	clinic := newClinic(t, cnpjFor(t, 10))
	if err := repo.Create(ctx, clinic); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, _ := repo.Get(ctx, clinic.ID)
	got.LegalName = "MUTAÇÃO EXTERNA"

	fresh, _ := repo.Get(ctx, clinic.ID)
	if fresh.LegalName != clinic.LegalName {
		t.Errorf("armazenamento afetado por mutação externa: %q", fresh.LegalName)
	}
}

func TestClinicRepoConcorrencia(t *testing.T) {
	t.Parallel()

	t.Run("criações distintas em paralelo", func(t *testing.T) {
		t.Parallel()
		repo := memory.NewClinicRepository()
		ctx := context.Background()
		const n = 30

		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := repo.Create(ctx, newClinic(t, cnpjFor(t, 100+i))); err != nil {
					t.Errorf("Create %d: %v", i, err)
				}
			}()
		}
		wg.Wait()

		if list, _ := repo.List(ctx); len(list) != n {
			t.Errorf("List = %d itens, quer %d", len(list), n)
		}
	})

	t.Run("mesmo documento em paralelo", func(t *testing.T) {
		t.Parallel()
		repo := memory.NewClinicRepository()
		ctx := context.Background()
		doc := cnpjFor(t, 200)
		const n = 20

		var wg sync.WaitGroup
		results := make(chan error, n)
		for range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- repo.Create(ctx, newClinic(t, doc))
			}()
		}
		wg.Wait()
		close(results)

		sucessos, duplicados := 0, 0
		for err := range results {
			switch {
			case err == nil:
				sucessos++
			case errors.Is(err, domain.ErrDocumentAlreadyExists):
				duplicados++
			default:
				t.Errorf("erro inesperado: %v", err)
			}
		}
		if sucessos != 1 || duplicados != n-1 {
			t.Errorf("sucessos = %d, duplicados = %d; quer 1 e %d", sucessos, duplicados, n-1)
		}
	})

	t.Run("leituras e escritas mistas", func(t *testing.T) {
		t.Parallel()
		repo := memory.NewClinicRepository()
		ctx := context.Background()

		clinic := newClinic(t, cnpjFor(t, 300))
		if err := repo.Create(ctx, clinic); err != nil {
			t.Fatalf("Create: %v", err)
		}

		var wg sync.WaitGroup
		for i := range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				switch i % 3 {
				case 0:
					_, _ = repo.Get(ctx, clinic.ID)
				case 1:
					_, _ = repo.List(ctx)
				default:
					_ = repo.Create(ctx, newClinic(t, cnpjFor(t, 400+i)))
				}
			}()
		}
		wg.Wait()
	})
}
