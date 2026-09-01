package service

import (
	"context"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// DentistInput são os dados cadastrais de um dentista, usados na criação
// e na alteração.
type DentistInput struct {
	Name  string
	Phone string
	Email string
	Admin bool
}

// DentistService implementa os casos de uso de gestão de dentistas,
// sempre no escopo de uma clínica ativa.
type DentistService struct {
	dentists port.DentistRepository
	clinics  port.ClinicRepository
}

// NewDentistService cria o service de dentistas.
func NewDentistService(dentists port.DentistRepository, clinics port.ClinicRepository) *DentistService {
	return &DentistService{dentists: dentists, clinics: clinics}
}

// Create valida e persiste um dentista vinculado a uma clínica ativa.
func (s *DentistService) Create(ctx context.Context, clinicID string, in DentistInput) (domain.Dentist, error) {
	if _, err := s.clinics.Get(ctx, clinicID); err != nil {
		return domain.Dentist{}, err
	}
	dentist, err := domain.NewDentist(clinicID, in.Name, in.Phone, in.Email, in.Admin)
	if err != nil {
		return domain.Dentist{}, err
	}
	if err := s.dentists.Create(ctx, *dentist); err != nil {
		return domain.Dentist{}, err
	}
	return *dentist, nil
}

// Get retorna um dentista ativo da clínica informada.
func (s *DentistService) Get(ctx context.Context, clinicID, dentistID string) (domain.Dentist, error) {
	return s.getScoped(ctx, clinicID, dentistID)
}

// ListByClinic retorna os dentistas ativos de uma clínica ativa.
func (s *DentistService) ListByClinic(ctx context.Context, clinicID string) ([]domain.Dentist, error) {
	if _, err := s.clinics.Get(ctx, clinicID); err != nil {
		return nil, err
	}
	return s.dentists.ListByClinic(ctx, clinicID)
}

// Update altera os dados cadastrais de um dentista ativo da clínica.
func (s *DentistService) Update(ctx context.Context, clinicID, dentistID string, in DentistInput) (domain.Dentist, error) {
	dentist, err := s.getScoped(ctx, clinicID, dentistID)
	if err != nil {
		return domain.Dentist{}, err
	}
	if err := dentist.Update(in.Name, in.Phone, in.Email, in.Admin); err != nil {
		return domain.Dentist{}, err
	}
	if err := s.dentists.Update(ctx, dentist); err != nil {
		return domain.Dentist{}, err
	}
	return dentist, nil
}

// Delete exclui um dentista (soft delete) da clínica informada.
func (s *DentistService) Delete(ctx context.Context, clinicID, dentistID string) error {
	if _, err := s.getScoped(ctx, clinicID, dentistID); err != nil {
		return err
	}
	return s.dentists.Delete(ctx, dentistID)
}

// getScoped responde not found para dentista de outra clínica, não
// vazando a existência de registros entre clínicas.
func (s *DentistService) getScoped(ctx context.Context, clinicID, dentistID string) (domain.Dentist, error) {
	dentist, err := s.dentists.Get(ctx, dentistID)
	if err != nil {
		return domain.Dentist{}, err
	}
	if dentist.ClinicID != clinicID {
		return domain.Dentist{}, domain.ErrNotFound
	}
	return dentist, nil
}
