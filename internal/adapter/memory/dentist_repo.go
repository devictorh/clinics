package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// DentistRepository implementa port.DentistRepository sobre map + RWMutex.
type DentistRepository struct {
	mu       sync.RWMutex
	dentists map[string]domain.Dentist
}

var _ port.DentistRepository = (*DentistRepository)(nil)

// NewDentistRepository cria um repositório de dentistas vazio.
func NewDentistRepository() *DentistRepository {
	return &DentistRepository{dentists: make(map[string]domain.Dentist)}
}

// Create persiste o dentista.
func (r *DentistRepository) Create(_ context.Context, dentist domain.Dentist) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.dentists[dentist.ID]; ok {
		return fmt.Errorf("%w: id já cadastrado", domain.ErrInvalidInput)
	}
	r.dentists[dentist.ID] = dentist
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
// é imutável.
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
	r.dentists[dentist.ID] = dentist
	return nil
}

// Delete marca o dentista como excluído.
func (r *DentistRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dentist, ok := r.dentists[id]
	if !ok || dentist.IsDeleted() {
		return domain.ErrNotFound
	}
	dentist.Delete()
	r.dentists[id] = dentist
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
		}
	}
	return nil
}
