package port

import (
	"context"

	"github.com/devictorh/clinics/internal/core/domain"
)

// PixChargeInput são os dados que o provedor precisa para gerar uma
// cobrança.
type PixChargeInput struct {
	Amount       domain.Amount
	MerchantName string
}

// PixProvider gera cobranças Pix, devolvendo o código copia-e-cola. Em
// produção seria um PSP real; no projeto, um simulador cumpre o contrato.
type PixProvider interface {
	GenerateCharge(ctx context.Context, in PixChargeInput) (string, error)
}
