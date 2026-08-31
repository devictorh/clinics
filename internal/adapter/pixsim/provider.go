package pixsim

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/devictorh/clinics/internal/core/port"
)

const (
	pixKey       = "recebimentos@clinics.simulado"
	merchantCity = "SAO PAULO"
	maxNameLen   = 25
	txidLen      = 25
)

// Provider implementa port.PixProvider gerando cobranças simuladas: um
// txid próprio e um código copia-e-cola no formato TLV do BR Code (EMV).
// O CRC final é fixo — o código é estruturalmente realista para exibição
// e integração de front-end, mas não é pagável.
type Provider struct{}

var _ port.PixProvider = (*Provider)(nil)

// NewProvider cria o provedor Pix simulado.
func NewProvider() *Provider { return &Provider{} }

// GenerateCharge monta o código copia-e-cola da cobrança.
func (p *Provider) GenerateCharge(_ context.Context, in port.PixChargeInput) (string, error) {
	txid := strings.ReplaceAll(uuid.NewString(), "-", "")[:txidLen]

	merchantAccount := tlv("00", "br.gov.bcb.pix") + tlv("01", pixKey)
	payload := tlv("00", "01") +
		tlv("26", merchantAccount) +
		tlv("52", "0000") +
		tlv("53", "986") +
		tlv("54", in.Amount.String()) +
		tlv("58", "BR") +
		tlv("59", merchantName(in.MerchantName)) +
		tlv("60", merchantCity) +
		tlv("62", tlv("05", txid))

	return payload + "6304FFFF", nil
}

// tlv codifica um campo do BR Code: id + comprimento com 2 dígitos + valor.
func tlv(id, value string) string {
	return fmt.Sprintf("%s%02d%s", id, len(value), value)
}

func merchantName(name string) string {
	runes := []rune(strings.ToUpper(strings.TrimSpace(name)))
	if len(runes) > maxNameLen {
		runes = runes[:maxNameLen]
	}
	return string(runes)
}
