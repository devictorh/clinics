package memory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// ClinicRepository implementa port.ClinicRepository sobre map + RWMutex.
// O índice de documento cobre apenas clínicas ativas: a unicidade é
// garantida de forma atômica sob o mesmo lock do map, e o soft delete
// libera o documento para novo cadastro.
type ClinicRepository struct {
	mu       sync.RWMutex
	clinics  map[string]domain.Clinic
	docIndex map[string]string
}

var _ port.ClinicRepository = (*ClinicRepository)(nil)

// NewClinicRepository cria um repositório de clínicas vazio.
func NewClinicRepository() *ClinicRepository {
	return &ClinicRepository{
		clinics:  make(map[string]domain.Clinic),
		docIndex: make(map[string]string),
	}
}

// Create persiste a clínica, garantindo unicidade de documento entre ativas.
func (r *ClinicRepository) Create(_ context.Context, clinic domain.Clinic) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.docIndex[clinic.Document.String()]; ok {
		return domain.ErrDocumentAlreadyExists
	}
	if _, ok := r.clinics[clinic.ID]; ok {
		return fmt.Errorf("%w: id já cadastrado", domain.ErrInvalidInput)
	}
	r.clinics[clinic.ID] = clinic
	r.docIndex[clinic.Document.String()] = clinic.ID
	return nil
}

// Get retorna a clínica ativa com o ID informado.
func (r *ClinicRepository) Get(_ context.Context, id string) (domain.Clinic, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clinic, ok := r.clinics[id]
	if !ok || clinic.IsDeleted() {
		return domain.Clinic{}, domain.ErrNotFound
	}
	return clinic, nil
}

// List retorna as clínicas ativas ordenadas por criação.
func (r *ClinicRepository) List(_ context.Context) ([]domain.Clinic, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clinics := make([]domain.Clinic, 0, len(r.clinics))
	for _, clinic := range r.clinics {
		if !clinic.IsDeleted() {
			clinics = append(clinics, clinic)
		}
	}
	sortByCreation(clinics, func(c domain.Clinic) (time.Time, string) {
		return c.CreatedAt, c.ID
	})
	return clinics, nil
}

// Update substitui os dados de uma clínica ativa; o documento é imutável.
func (r *ClinicRepository) Update(_ context.Context, clinic domain.Clinic) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.clinics[clinic.ID]
	if !ok || current.IsDeleted() {
		return domain.ErrNotFound
	}
	if current.Document != clinic.Document {
		return fmt.Errorf("%w: documento é imutável", domain.ErrInvalidInput)
	}
	r.clinics[clinic.ID] = clinic
	return nil
}

// Delete marca a clínica como excluída e libera seu documento.
func (r *ClinicRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clinic, ok := r.clinics[id]
	if !ok || clinic.IsDeleted() {
		return domain.ErrNotFound
	}
	clinic.Delete()
	r.clinics[id] = clinic
	delete(r.docIndex, clinic.Document.String())
	return nil
}

// sortByCreation ordena por instante de criação com desempate por ID,
// garantindo resultado determinístico para iteração de map.
func sortByCreation[T any](items []T, key func(T) (time.Time, string)) {
	slices.SortFunc(items, func(a, b T) int {
		aTime, aID := key(a)
		bTime, bID := key(b)
		if c := aTime.Compare(bTime); c != 0 {
			return c
		}
		return strings.Compare(aID, bID)
	})
}
