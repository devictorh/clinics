package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Dentist representa um dentista, sempre vinculado a uma clínica. A flag
// Admin marca o dentista como administrador e responsável legal da
// clínica; uma clínica pode ter um ou mais administradores.
type Dentist struct {
	ID        string
	ClinicID  string
	Name      string
	Phone     string
	Email     string
	Admin     bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// NewDentist cria um dentista válido vinculado à clínica informada. O
// telefone é normalizado para apenas dígitos.
func NewDentist(clinicID, name, phone, email string, admin bool) (*Dentist, error) {
	if strings.TrimSpace(clinicID) == "" {
		return nil, invalidInput("clínica é obrigatória")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalidInput("nome é obrigatório")
	}
	normalizedPhone, err := normalizePhone(phone)
	if err != nil {
		return nil, err
	}
	email, err = validateEmail(email)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Dentist{
		ID:        uuid.NewString(),
		ClinicID:  clinicID,
		Name:      name,
		Phone:     normalizedPhone,
		Email:     email,
		Admin:     admin,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update altera os dados cadastrais do dentista.
func (d *Dentist) Update(name, phone, email string, admin bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return invalidInput("nome é obrigatório")
	}
	normalizedPhone, err := normalizePhone(phone)
	if err != nil {
		return err
	}
	email, err = validateEmail(email)
	if err != nil {
		return err
	}
	d.Name = name
	d.Phone = normalizedPhone
	d.Email = email
	d.Admin = admin
	d.touch()
	return nil
}

// Delete marca o dentista como excluído (soft delete). É idempotente: um
// dentista já excluído mantém o instante da primeira exclusão.
func (d *Dentist) Delete() {
	if d.DeletedAt != nil {
		return
	}
	now := time.Now().UTC()
	d.DeletedAt = &now
	d.UpdatedAt = now
}

// IsDeleted indica se o dentista foi excluído via soft delete.
func (d *Dentist) IsDeleted() bool { return d.DeletedAt != nil }

func (d *Dentist) touch() { d.UpdatedAt = time.Now().UTC() }

// normalizePhone valida um telefone brasileiro de forma pragmática:
// remove a formatação usual e exige 10 (fixo) ou 11 (celular) dígitos,
// com DDD. Qualquer outro caractere é rejeitado.
func normalizePhone(raw string) (string, error) {
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c >= '0' && c <= '9':
			b.WriteByte(c)
		case c == '(' || c == ')' || c == '-' || c == ' ':
		default:
			return "", invalidInput("telefone contém caracteres inválidos")
		}
	}
	digits := b.String()
	if len(digits) < 10 || len(digits) > 11 {
		return "", invalidInput("telefone deve ter 10 ou 11 dígitos, com DDD")
	}
	return digits, nil
}

// validateEmail valida o email de forma pragmática via net/mail,
// rejeitando formas com display name (ex.: "Nome <a@b.com>").
func validateEmail(raw string) (string, error) {
	email := strings.TrimSpace(raw)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", invalidInput("email inválido")
	}
	return email, nil
}
