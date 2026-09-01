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
// soft delete da clínica. O email é único entre os dentistas ativos de
// uma mesma clínica (case-insensitive) — o mesmo profissional pode
// existir em clínicas diferentes.
type DentistRepository interface {
	Create(ctx context.Context, dentist domain.Dentist) error
	Get(ctx context.Context, id string) (domain.Dentist, error)
	ListByClinic(ctx context.Context, clinicID string) ([]domain.Dentist, error)
	Update(ctx context.Context, dentist domain.Dentist) error
	Delete(ctx context.Context, id string) error
	DeleteByClinicID(ctx context.Context, clinicID string) error
}

// PaymentRepository persiste cobranças, que são histórico financeiro
// imutável: não há update nem delete, e a única mutação é a transição de
// status feita atomicamente por Approve, que devolve o registro
// atualizado. Mesma semântica de cópias dos demais repositórios.
type PaymentRepository interface {
	Create(ctx context.Context, payment domain.Payment) error
	Get(ctx context.Context, id string) (domain.Payment, error)
	ListByClinic(ctx context.Context, clinicID string) ([]domain.Payment, error)
	Approve(ctx context.Context, id string) (domain.Payment, error)
}
