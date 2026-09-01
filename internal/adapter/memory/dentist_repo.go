package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// DentistRepository implementa port.DentistRepository sobre map + RWMutex.
// O índice de email cobre apenas dentistas ativos, por clínica: a
// unicidade é garantida de forma atômica sob o mesmo lock do map, o soft
// delete libera o email para recadastro na clínica, e o mesmo email pode
// existir em clínicas diferentes.
type DentistRepository struct {
	mu         sync.RWMutex
	dentists   map[string]domain.Dentist
	emailIndex map[string]string
}

var _ port.DentistRepository = (*DentistRepository)(nil)

// NewDentistRepository cria um repositório de dentistas vazio.
func NewDentistRepository() *DentistRepository {
	return &DentistRepository{
		dentists:   make(map[string]domain.Dentist),
		emailIndex: make(map[string]string),
	}
}

// Create persiste o dentista, garantindo unicidade de email entre os
// ativos da clínica.
func (r *DentistRepository) Create(_ context.Context, dentist domain.Dentist) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.dentists[dentist.ID]; ok {
		return fmt.Errorf("%w: id já cadastrado", domain.ErrInvalidInput)
	}
	key := emailKey(dentist.ClinicID, dentist.Email)
	if _, ok := r.emailIndex[key]; ok {
		return domain.ErrEmailAlreadyExists
	}
	r.dentists[dentist.ID] = dentist
	r.emailIndex[key] = dentist.ID
	return nil
}

// Get retorna o dentista ativo com o ID informado.
func (r *DentistRepository) Get(_ context.Context, id string) (domain.Dentist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dentist, ok := r.dentists[id]
	if !ok || dentist.IsDeleted() {
		return domain.Dentist{}, domain.ErrNotFound
	}
	return dentist, nil
}

// ListByClinic retorna os dentistas ativos da clínica ordenados por criação.
func (r *DentistRepository) ListByClinic(_ context.Context, clinicID string) ([]domain.Dentist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dentists := make([]domain.Dentist, 0)
	for _, dentist := range r.dentists {
		if dentist.ClinicID == clinicID && !dentist.IsDeleted() {
			dentists = append(dentists, dentist)
		}
	}
	sortByCreation(dentists, func(d domain.Dentist) (time.Time, string) {
		return d.CreatedAt, d.ID
	})
	return dentists, nil
}

// Update substitui os dados de um dentista ativo; o vínculo com a clínica
// é imutável e o novo email não pode colidir com outro dentista ativo da
// clínica.
func (r *DentistRepository) Update(_ context.Context, dentist domain.Dentist) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.dentists[dentist.ID]
	if !ok || current.IsDeleted() {
		return domain.ErrNotFound
	}
	if current.ClinicID != dentist.ClinicID {
		return fmt.Errorf("%w: vínculo com a clínica é imutável", domain.ErrInvalidInput)
	}

	oldKey := emailKey(current.ClinicID, current.Email)
	newKey := emailKey(dentist.ClinicID, dentist.Email)
	if newKey != oldKey {
		if _, ok := r.emailIndex[newKey]; ok {
			return domain.ErrEmailAlreadyExists
		}
		delete(r.emailIndex, oldKey)
		r.emailIndex[newKey] = dentist.ID
	}
	r.dentists[dentist.ID] = dentist
	return nil
}

// Delete marca o dentista como excluído e libera seu email na clínica.
func (r *DentistRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dentist, ok := r.dentists[id]
	if !ok || dentist.IsDeleted() {
		return domain.ErrNotFound
	}
	dentist.Delete()
	r.dentists[id] = dentist
	delete(r.emailIndex, emailKey(dentist.ClinicID, dentist.Email))
	return nil
}

// DeleteByClinicID marca como excluídos todos os dentistas ativos da clínica.
func (r *DentistRepository) DeleteByClinicID(_ context.Context, clinicID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, dentist := range r.dentists {
		if dentist.ClinicID == clinicID && !dentist.IsDeleted() {
			dentist.Delete()
			r.dentists[id] = dentist
			delete(r.emailIndex, emailKey(dentist.ClinicID, dentist.Email))
		}
	}
	return nil
}

func emailKey(clinicID, email string) string {
	return clinicID + "|" + strings.ToLower(email)
}
