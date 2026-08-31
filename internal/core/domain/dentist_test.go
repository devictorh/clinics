package domain_test

import (
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
)

func newDentist(t *testing.T) *domain.Dentist {
	t.Helper()
	d, err := domain.NewDentist("clinic-1", "Dra. Ana Souza", "(11) 98765-4321", "ana@sorriso.com.br", true)
	if err != nil {
		t.Fatalf("NewDentist: %v", err)
	}
	return d
}

func TestNewDentist(t *testing.T) {
	t.Parallel()

	d := newDentist(t)

	if d.ID == "" {
		t.Error("ID não gerado")
	}
	if d.ClinicID != "clinic-1" {
		t.Errorf("ClinicID = %q", d.ClinicID)
	}
	if d.Phone != "11987654321" {
		t.Errorf("telefone não normalizado: %q", d.Phone)
	}
	if !d.Admin {
		t.Error("flag admin não preservada")
	}
	if d.IsDeleted() {
		t.Error("dentista novo não deveria estar excluído")
	}
}

func TestNewDentistInvalido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                          string
		clinicID, dName, phone, email string
	}{
		{"clínica vazia", "", "Ana", "(11) 98765-4321", "ana@x.com"},
		{"nome vazio", "clinic-1", "  ", "(11) 98765-4321", "ana@x.com"},
		{"telefone curto", "clinic-1", "Ana", "987654321", "ana@x.com"},
		{"telefone longo", "clinic-1", "Ana", "5511987654321", "ana@x.com"},
		{"telefone com letras", "clinic-1", "Ana", "11abc543210", "ana@x.com"},
		{"email vazio", "clinic-1", "Ana", "(11) 98765-4321", ""},
		{"email sem arroba", "clinic-1", "Ana", "(11) 98765-4321", "ana.x.com"},
		{"email com display name", "clinic-1", "Ana", "(11) 98765-4321", "Ana <ana@x.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewDentist(tt.clinicID, tt.dName, tt.phone, tt.email, false)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("erro = %v, quer ErrInvalidInput", err)
			}
		})
	}
}

func TestDentistUpdate(t *testing.T) {
	t.Parallel()

	d := newDentist(t)

	if err := d.Update("Dr. Bruno Lima", "11 3265-4321", "bruno@sorriso.com.br", false); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if d.Name != "Dr. Bruno Lima" || d.Phone != "1132654321" || d.Email != "bruno@sorriso.com.br" {
		t.Errorf("dados após update = %q / %q / %q", d.Name, d.Phone, d.Email)
	}
	if d.Admin {
		t.Error("flag admin deveria ter sido desligada")
	}

	if err := d.Update("Bruno", "123", "bruno@sorriso.com.br", false); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("update inválido: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestDentistDelete(t *testing.T) {
	t.Parallel()

	d := newDentist(t)

	d.Delete()
	if !d.IsDeleted() || d.DeletedAt == nil {
		t.Fatal("dentista deveria estar excluído")
	}

	first := *d.DeletedAt
	d.Delete()
	if !d.DeletedAt.Equal(first) {
		t.Error("Delete não é idempotente: instante da exclusão mudou")
	}
}
