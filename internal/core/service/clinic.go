package service

import (
	"context"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// BankAccountInput são os dados bancários recebidos da borda.
type BankAccountInput struct {
	Bank    string
	Agency  string
	Account string
}

// CreateClinicInput são os dados de criação de uma clínica; os dados
// bancários são opcionais e podem ser definidos depois.
type CreateClinicInput struct {
	Document    string
	LegalName   string
	TradeName   string
	BankAccount *BankAccountInput
}

// UpdateClinicInput são os dados cadastrais alteráveis de uma clínica.
type UpdateClinicInput struct {
	LegalName string
	TradeName string
}

// ClinicService implementa os casos de uso de gestão de clínicas.
type ClinicService struct {
	clinics  port.ClinicRepository
	dentists port.DentistRepository
}

// NewClinicService cria o service de clínicas.
func NewClinicService(clinics port.ClinicRepository, dentists port.DentistRepository) *ClinicService {
	return &ClinicService{clinics: clinics, dentists: dentists}
}

// Create valida e persiste uma clínica nova.
func (s *ClinicService) Create(ctx context.Context, in CreateClinicInput) (domain.Clinic, error) {
	doc, err := domain.NewDocument(in.Document)
	if err != nil {
		return domain.Clinic{}, err
	}
	clinic, err := domain.NewClinic(doc, in.LegalName, in.TradeName)
	if err != nil {
		return domain.Clinic{}, err
	}
	if in.BankAccount != nil {
		account, err := domain.NewBankAccount(in.BankAccount.Bank, in.BankAccount.Agency, in.BankAccount.Account)
		if err != nil {
			return domain.Clinic{}, err
		}
		if err := clinic.UpdateBankAccount(account); err != nil {
			return domain.Clinic{}, err
		}
	}
	if err := s.clinics.Create(ctx, *clinic); err != nil {
		return domain.Clinic{}, err
	}
	return *clinic, nil
}

// Get retorna uma clínica ativa.
func (s *ClinicService) Get(ctx context.Context, id string) (domain.Clinic, error) {
	return s.clinics.Get(ctx, id)
}

// List retorna as clínicas ativas.
func (s *ClinicService) List(ctx context.Context) ([]domain.Clinic, error) {
	return s.clinics.List(ctx)
}

// Update altera os dados cadastrais de uma clínica ativa.
func (s *ClinicService) Update(ctx context.Context, id string, in UpdateClinicInput) (domain.Clinic, error) {
	clinic, err := s.clinics.Get(ctx, id)
	if err != nil {
		return domain.Clinic{}, err
	}
	if err := clinic.Update(in.LegalName, in.TradeName); err != nil {
		return domain.Clinic{}, err
	}
	if err := s.clinics.Update(ctx, clinic); err != nil {
		return domain.Clinic{}, err
	}
	return clinic, nil
}

// UpdateBankAccount define ou substitui os dados bancários de uma clínica
// ativa.
func (s *ClinicService) UpdateBankAccount(ctx context.Context, id string, in BankAccountInput) (domain.Clinic, error) {
	clinic, err := s.clinics.Get(ctx, id)
	if err != nil {
		return domain.Clinic{}, err
	}
	account, err := domain.NewBankAccount(in.Bank, in.Agency, in.Account)
	if err != nil {
		return domain.Clinic{}, err
	}
	if err := clinic.UpdateBankAccount(account); err != nil {
		return domain.Clinic{}, err
	}
	if err := s.clinics.Update(ctx, clinic); err != nil {
		return domain.Clinic{}, err
	}
	return clinic, nil
}

// Delete exclui a clínica (soft delete) e, em cascata, seus dentistas.
func (s *ClinicService) Delete(ctx context.Context, id string) error {
	if err := s.clinics.Delete(ctx, id); err != nil {
		return err
	}
	return s.dentists.DeleteByClinicID(ctx, id)
}
