package domain_test

import (
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
)

func newClinic(t *testing.T) *domain.Clinic {
	t.Helper()
	c, err := domain.NewClinic(mustDocument(t, "11222333000181"), "Clínica Sorriso LTDA", "Sorriso Odonto")
	if err != nil {
		t.Fatalf("NewClinic: %v", err)
	}
	return c
}

func TestNewClinic(t *testing.T) {
	t.Parallel()

	c := newClinic(t)

	if c.ID == "" {
		t.Error("ID não gerado")
	}
	if c.Document.String() != "11222333000181" {
		t.Errorf("documento = %q", c.Document.String())
	}
	if c.LegalName != "Clínica Sorriso LTDA" || c.TradeName != "Sorriso Odonto" {
		t.Errorf("nomes = %q / %q", c.LegalName, c.TradeName)
	}
	if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
		t.Error("timestamps não definidos")
	}
	if c.IsDeleted() {
		t.Error("clínica nova não deveria estar excluída")
	}
	if !c.BankAccount.IsZero() {
		t.Error("clínica nova não deveria ter dados bancários")
	}
}

func TestNewClinicInvalida(t *testing.T) {
	t.Parallel()

	doc := mustDocument(t, "52998224725")
	tests := []struct {
		name      string
		doc       domain.Document
		legalName string
		tradeName string
	}{
		{"documento zero", domain.Document{}, "Razão", "Fantasia"},
		{"razão social vazia", doc, "", "Fantasia"},
		{"razão social só espaços", doc, "   ", "Fantasia"},
		{"nome fantasia vazio", doc, "Razão", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewClinic(tt.doc, tt.legalName, tt.tradeName)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("erro = %v, quer ErrInvalidInput", err)
			}
		})
	}
}

func TestClinicUpdate(t *testing.T) {
	t.Parallel()

	c := newClinic(t)

	if err := c.Update("Nova Razão LTDA", "Novo Fantasia"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c.LegalName != "Nova Razão LTDA" || c.TradeName != "Novo Fantasia" {
		t.Errorf("nomes após update = %q / %q", c.LegalName, c.TradeName)
	}
	if c.UpdatedAt.Before(c.CreatedAt) {
		t.Error("UpdatedAt não avançou")
	}

	if err := c.Update("", "Fantasia"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("update inválido: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestNewBankAccount(t *testing.T) {
	t.Parallel()

	if _, err := domain.NewBankAccount("341", "0001", "12345-6"); err != nil {
		t.Fatalf("NewBankAccount válida: %v", err)
	}

	tests := []struct{ name, bank, agency, account string }{
		{"banco vazio", "", "0001", "12345-6"},
		{"agência vazia", "341", "", "12345-6"},
		{"conta vazia", "341", "0001", ""},
		{"só espaços", " ", " ", " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewBankAccount(tt.bank, tt.agency, tt.account)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("erro = %v, quer ErrInvalidInput", err)
			}
		})
	}
}

func TestClinicUpdateBankAccount(t *testing.T) {
	t.Parallel()

	c := newClinic(t)

	ba, err := domain.NewBankAccount("341", "0001", "12345-6")
	if err != nil {
		t.Fatalf("NewBankAccount: %v", err)
	}
	if err := c.UpdateBankAccount(ba); err != nil {
		t.Fatalf("UpdateBankAccount: %v", err)
	}
	if c.BankAccount != ba {
		t.Errorf("BankAccount = %+v, quer %+v", c.BankAccount, ba)
	}

	if err := c.UpdateBankAccount(domain.BankAccount{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("conta zero: erro = %v, quer ErrInvalidInput", err)
	}
}

func TestClinicDelete(t *testing.T) {
	t.Parallel()

	c := newClinic(t)

	c.Delete()
	if !c.IsDeleted() || c.DeletedAt == nil {
		t.Fatal("clínica deveria estar excluída")
	}

	first := *c.DeletedAt
	c.Delete()
	if !c.DeletedAt.Equal(first) {
		t.Error("Delete não é idempotente: instante da exclusão mudou")
	}
}
