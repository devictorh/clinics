package domain_test

import (
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
)

func mustDocument(t *testing.T, raw string) domain.Document {
	t.Helper()
	d, err := domain.NewDocument(raw)
	if err != nil {
		t.Fatalf("NewDocument(%q): %v", raw, err)
	}
	return d
}

func TestNewDocumentValidos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantValue string
		wantType  domain.DocumentType
		wantMask  string
	}{
		{"cpf com máscara", "529.982.247-25", "52998224725", domain.DocumentTypeCPF, "529.982.247-25"},
		{"cpf sem máscara", "52998224725", "52998224725", domain.DocumentTypeCPF, "529.982.247-25"},
		{"cpf com espaços", "  529 982 247 25 ", "52998224725", domain.DocumentTypeCPF, "529.982.247-25"},
		{"cpf clássico de dígitos repetidos na base", "11144477735", "11144477735", domain.DocumentTypeCPF, "111.444.777-35"},

		{"cnpj numérico com máscara", "11.222.333/0001-81", "11222333000181", domain.DocumentTypeCNPJ, "11.222.333/0001-81"},
		{"cnpj numérico sem máscara", "11222333000181", "11222333000181", domain.DocumentTypeCNPJ, "11.222.333/0001-81"},

		// Exemplo oficial de CNPJ alfanumérico divulgado pela Receita/Serpro.
		{"cnpj alfanumérico", "12ABC34501DE35", "12ABC34501DE35", domain.DocumentTypeCNPJ, "12.ABC.345/01DE-35"},
		{"cnpj alfanumérico com máscara", "12.ABC.345/01DE-35", "12ABC34501DE35", domain.DocumentTypeCNPJ, "12.ABC.345/01DE-35"},
		{"cnpj alfanumérico minúsculo", "12abc34501de35", "12ABC34501DE35", domain.DocumentTypeCNPJ, "12.ABC.345/01DE-35"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := domain.NewDocument(tt.input)
			if err != nil {
				t.Fatalf("NewDocument(%q) erro inesperado: %v", tt.input, err)
			}
			if d.String() != tt.wantValue {
				t.Errorf("valor = %q, quer %q", d.String(), tt.wantValue)
			}
			if d.Type() != tt.wantType {
				t.Errorf("tipo = %q, quer %q", d.Type(), tt.wantType)
			}
			if d.Formatted() != tt.wantMask {
				t.Errorf("Formatted() = %q, quer %q", d.Formatted(), tt.wantMask)
			}
			if d.IsZero() {
				t.Error("IsZero() = true para documento válido")
			}
		})
	}
}

func TestNewDocumentInvalidos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"vazio", ""},
		{"só separadores", ".-/ "},
		{"cpf curto", "1234567890"},
		{"tamanho intermediário", "123456789012"},
		{"cpf dv errado", "52998224724"},
		{"cpf todos iguais", "11111111111"},
		{"cpf zerado", "00000000000"},
		{"cpf com letra", "5299822472A"},
		{"cnpj dv errado", "11222333000182"},
		{"cnpj zerado", "00000000000000"},
		{"cnpj com letra no dv", "11222333000A81"},
		{"cnpj com letra nos dois dv", "12ABC34501DEAB"},
		{"cnpj alfanumérico dv errado", "12ABC34501DE36"},
		{"caractere inválido", "12ABC34501D#35"},
		{"acentuado", "12ÁBC34501DE35"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := domain.NewDocument(tt.input)
			if err == nil {
				t.Fatalf("NewDocument(%q) devia falhar, retornou %q", tt.input, d.String())
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("erro não encapsula ErrInvalidInput: %v", err)
			}
			if !d.IsZero() {
				t.Errorf("documento devia ser zero em caso de erro, veio %q", d.String())
			}
		})
	}
}

func TestDocumentZeroValue(t *testing.T) {
	t.Parallel()

	var d domain.Document
	if !d.IsZero() {
		t.Error("Document{} devia ser zero")
	}
	if d.Formatted() != "" {
		t.Errorf("Formatted() do zero = %q, quer vazio", d.Formatted())
	}
}

func TestDocumentComparavel(t *testing.T) {
	t.Parallel()

	a := mustDocument(t, "11.222.333/0001-81")
	b := mustDocument(t, "11222333000181")
	if a != b {
		t.Error("mesmo documento com máscaras diferentes devia ser igual após normalização")
	}
}

// TestDocumentDVCoerencia confere que o cálculo aceita exatamente um par de
// dígitos verificadores por base: qualquer outro par deve ser rejeitado.
func TestDocumentDVCoerencia(t *testing.T) {
	t.Parallel()

	bases := []string{"12ABC34501DE", "112223330001", "ZZ99AA88BB77", "529982247"}
	for _, base := range bases {
		aceitos := 0
		for i := range 100 {
			cand := base + string([]byte{byte('0' + i/10), byte('0' + i%10)})
			if _, err := domain.NewDocument(cand); err == nil {
				aceitos++
			}
		}
		if aceitos != 1 {
			t.Errorf("base %s aceitou %d pares de DV, esperava 1", base, aceitos)
		}
	}
}
