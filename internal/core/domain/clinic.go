package domain

import (
	"strings"
	"time"
	"uuid"
)

// BankAccount agrupa os dados bancários de recebimento de uma clínica.
// Um BankAccount não-zero é sempre completo: os três campos são validados
// juntos na construção.
type BankAccount struct {
	Bank    string
	Agency  string
	Account string
}

// NewBankAccount valida os dados bancários; banco, agência e conta são
// obrigatórios.
func NewBankAccount(bank, agency, account string) (BankAccount, error) {
	bank = strings.TrimSpace(bank)
	agency = strings.TrimSpace(agency)
	account = strings.TrimSpace(account)
	if bank == "" || agency == "" || account == "" {
		return BankAccount{}, invalidInput("banco, agência e conta são obrigatórios")
	}
	return BankAccount{Bank: bank, Agency: agency, Account: account}, nil
}

// IsZero indica ausência de dados bancários.
func (b BankAccount) IsZero() bool { return b == BankAccount{} }

// Clinic representa uma clínica odontológica. O documento é imutável após
// a criação — é a identidade fiscal do cadastro; os demais dados são
// alteráveis. Os dados bancários são opcionais na criação e definidos ou
// alterados via UpdateBankAccount.
type Clinic struct {
	ID          string
	Document    Document
	LegalName   string
	TradeName   string
	BankAccount BankAccount
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// NewClinic cria uma clínica válida com documento, razão social e nome
// fantasia obrigatórios.
func NewClinic(doc Document, legalName, tradeName string) (*Clinic, error) {
	legalName = strings.TrimSpace(legalName)
	tradeName = strings.TrimSpace(tradeName)
	switch {
	case doc.IsZero():
		return nil, invalidInput("documento é obrigatório")
	case legalName == "":
		return nil, invalidInput("razão social é obrigatória")
	case tradeName == "":
		return nil, invalidInput("nome fantasia é obrigatório")
	}

	now := time.Now().UTC()
	return &Clinic{
		ID:        uuid.New().String(),
		Document:  doc,
		LegalName: legalName,
		TradeName: tradeName,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update altera os dados cadastrais mutáveis da clínica.
func (c *Clinic) Update(legalName, tradeName string) error {
	legalName = strings.TrimSpace(legalName)
	tradeName = strings.TrimSpace(tradeName)
	if legalName == "" {
		return invalidInput("razão social é obrigatória")
	}
	if tradeName == "" {
		return invalidInput("nome fantasia é obrigatório")
	}
	c.LegalName = legalName
	c.TradeName = tradeName
	c.touch()
	return nil
}

// UpdateBankAccount define ou substitui os dados bancários da clínica.
func (c *Clinic) UpdateBankAccount(b BankAccount) error {
	if b.IsZero() {
		return invalidInput("dados bancários são obrigatórios")
	}
	c.BankAccount = b
	c.touch()
	return nil
}

// Delete marca a clínica como excluída (soft delete). É idempotente: uma
// clínica já excluída mantém o instante da primeira exclusão.
func (c *Clinic) Delete() {
	if c.DeletedAt != nil {
		return
	}
	now := time.Now().UTC()
	c.DeletedAt = &now
	c.UpdatedAt = now
}

// IsDeleted indica se a clínica foi excluída via soft delete.
func (c *Clinic) IsDeleted() bool { return c.DeletedAt != nil }

func (c *Clinic) touch() { c.UpdatedAt = time.Now().UTC() }
