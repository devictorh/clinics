package port

import (
	"context"

	"github.com/devictorh/clinics/internal/core/domain"
)

// ClinicRepository persiste clínicas com semântica de soft delete:
// registros excluídos permanecem armazenados mas ficam invisíveis às
// leituras, e a unicidade de documento vale apenas entre clínicas ativas.
// Os métodos recebem e devolvem cópias — mutações nos valores retornados
// não afetam o armazenamento.
type ClinicRepository interface {
	Create(ctx context.Context, clinic domain.Clinic) error
	Get(ctx context.Context, id string) (domain.Clinic, error)
	List(ctx context.Context) ([]domain.Clinic, error)
	Update(ctx context.Context, clinic domain.Clinic) error
	Delete(ctx context.Context, id string) error
}

// DentistRepository persiste dentistas com a mesma semântica de soft
// delete e de cópias do ClinicRepository; DeleteByClinicID é a cascata do
// soft delete da clínica.
type DentistRepository interface {
	Create(ctx context.Context, dentist domain.Dentist) error
	Get(ctx context.Context, id string) (domain.Dentist, error)
	ListByClinic(ctx context.Context, clinicID string) ([]domain.Dentist, error)
	Update(ctx context.Context, dentist domain.Dentist) error
	Delete(ctx context.Context, id string) error
	DeleteByClinicID(ctx context.Context, clinicID string) error
}
