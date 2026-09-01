package domain_test

import (
	"errors"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
)

func TestNewAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cents   int64
		wantErr bool
	}{
		{"um centavo", 1, false},
		{"valor comum", 15000, false},
		{"zero", 0, true},
		{"negativo", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := domain.NewAmount(tt.cents)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewAmount(%d) devia falhar", tt.cents)
				}
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Errorf("erro não encapsula ErrInvalidInput: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewAmount(%d) erro inesperado: %v", tt.cents, err)
			}
			if a.Cents() != tt.cents {
				t.Errorf("Cents() = %d, quer %d", a.Cents(), tt.cents)
			}
		})
	}
}

func TestAmountString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cents int64
		want  string
	}{
		{5, "0.05"},
		{50, "0.50"},
		{100, "1.00"},
		{1234, "12.34"},
		{15000, "150.00"},
	}

	for _, tt := range tests {
		a, err := domain.NewAmount(tt.cents)
		if err != nil {
			t.Fatalf("NewAmount(%d): %v", tt.cents, err)
		}
		if got := a.String(); got != tt.want {
			t.Errorf("String() de %d centavos = %q, quer %q", tt.cents, got, tt.want)
		}
	}
}
