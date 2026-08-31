package pixsim_test

import (
	"context"
	"strings"
	"testing"

	"github.com/devictorh/clinics/internal/adapter/pixsim"
	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

func generate(t *testing.T, merchant string) string {
	t.Helper()
	amount, err := domain.NewAmount(15000)
	if err != nil {
		t.Fatalf("NewAmount: %v", err)
	}
	code, err := pixsim.NewProvider().GenerateCharge(context.Background(), port.PixChargeInput{
		Amount:       amount,
		MerchantName: merchant,
	})
	if err != nil {
		t.Fatalf("GenerateCharge: %v", err)
	}
	return code
}

func TestGenerateCharge(t *testing.T) {
	t.Parallel()

	code := generate(t, "Sorriso Odonto")

	for _, want := range []string{
		"000201",             // payload format indicator
		"br.gov.bcb.pix",     // GUI da conta do recebedor
		"5406150.00",         // valor com duas casas (tag 54)
		"5802BR",             // país
		"5914SORRISO ODONTO", // nome em maiúsculas (tag 59)
	} {
		if !strings.Contains(code, want) {
			t.Errorf("código não contém %q: %s", want, code)
		}
	}
	if !strings.HasSuffix(code, "6304FFFF") {
		t.Errorf("código não termina com CRC simulado: %s", code)
	}
}

func TestGenerateChargeNomeLongo(t *testing.T) {
	t.Parallel()

	code := generate(t, "Clínica Odontológica com Nome Extremamente Longo LTDA")
	if strings.Contains(code, "EXTREMAMENTE") {
		t.Errorf("nome do recebedor não foi truncado: %s", code)
	}
}

func TestGenerateChargeTxidUnico(t *testing.T) {
	t.Parallel()

	primeiro := generate(t, "Sorriso")
	segundo := generate(t, "Sorriso")
	if primeiro == segundo {
		t.Error("duas cobranças geraram códigos idênticos — txid não é único")
	}
}
