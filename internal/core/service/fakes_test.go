package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/port"
)

// Fakes manuais dos ports: comportamento mínimo sobre map, com campos de
// erro para forçar falhas por método.

type fakeClinicRepo struct {
	items map[string]domain.Clinic

	createErr, getErr, listErr, updateErr, deleteErr error
}

var _ port.ClinicRepository = (*fakeClinicRepo)(nil)

func newFakeClinicRepo() *fakeClinicRepo {
	return &fakeClinicRepo{items: make(map[string]domain.Clinic)}
}

func (f *fakeClinicRepo) Create(_ context.Context, clinic domain.Clinic) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.items[clinic.ID] = clinic
	return nil
}

func (f *fakeClinicRepo) Get(_ context.Context, id string) (domain.Clinic, error) {
	if f.getErr != nil {
		return domain.Clinic{}, f.getErr
	}
	clinic, ok := f.items[id]
	if !ok || clinic.IsDeleted() {
		return domain.Clinic{}, domain.ErrNotFound
	}
	return clinic, nil
}

func (f *fakeClinicRepo) List(_ context.Context) ([]domain.Clinic, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	list := make([]domain.Clinic, 0, len(f.items))
	for _, clinic := range f.items {
		if !clinic.IsDeleted() {
			list = append(list, clinic)
		}
	}
	return list, nil
}

func (f *fakeClinicRepo) Update(_ context.Context, clinic domain.Clinic) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	current, ok := f.items[clinic.ID]
	if !ok || current.IsDeleted() {
		return domain.ErrNotFound
	}
	f.items[clinic.ID] = clinic
	return nil
}

func (f *fakeClinicRepo) Delete(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	clinic, ok := f.items[id]
	if !ok || clinic.IsDeleted() {
		return domain.ErrNotFound
	}
	clinic.Delete()
	f.items[id] = clinic
	return nil
}

type fakeDentistRepo struct {
	items        map[string]domain.Dentist
	cascadeCalls []string

	createErr, getErr, listErr, updateErr, deleteErr, cascadeErr error
}

var _ port.DentistRepository = (*fakeDentistRepo)(nil)

func newFakeDentistRepo() *fakeDentistRepo {
	return &fakeDentistRepo{items: make(map[string]domain.Dentist)}
}

func (f *fakeDentistRepo) Create(_ context.Context, dentist domain.Dentist) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.items[dentist.ID] = dentist
	return nil
}

func (f *fakeDentistRepo) Get(_ context.Context, id string) (domain.Dentist, error) {
	if f.getErr != nil {
		return domain.Dentist{}, f.getErr
	}
	dentist, ok := f.items[id]
	if !ok || dentist.IsDeleted() {
		return domain.Dentist{}, domain.ErrNotFound
	}
	return dentist, nil
}

func (f *fakeDentistRepo) ListByClinic(_ context.Context, clinicID string) ([]domain.Dentist, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	list := make([]domain.Dentist, 0)
	for _, dentist := range f.items {
		if dentist.ClinicID == clinicID && !dentist.IsDeleted() {
			list = append(list, dentist)
		}
	}
	return list, nil
}

func (f *fakeDentistRepo) Update(_ context.Context, dentist domain.Dentist) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	current, ok := f.items[dentist.ID]
	if !ok || current.IsDeleted() {
		return domain.ErrNotFound
	}
	f.items[dentist.ID] = dentist
	return nil
}

func (f *fakeDentistRepo) Delete(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	dentist, ok := f.items[id]
	if !ok || dentist.IsDeleted() {
		return domain.ErrNotFound
	}
	dentist.Delete()
	f.items[id] = dentist
	return nil
}

func (f *fakeDentistRepo) DeleteByClinicID(_ context.Context, clinicID string) error {
	if f.cascadeErr != nil {
		return f.cascadeErr
	}
	f.cascadeCalls = append(f.cascadeCalls, clinicID)
	for id, dentist := range f.items {
		if dentist.ClinicID == clinicID && !dentist.IsDeleted() {
			dentist.Delete()
			f.items[id] = dentist
		}
	}
	return nil
}

// fakePaymentRepo é sincronizado porque o worker de aprovação do
// PaymentService o acessa em goroutine própria.
type fakePaymentRepo struct {
	mu    sync.Mutex
	items map[string]domain.Payment

	createErr, getErr, listErr, approveErr error
}

var _ port.PaymentRepository = (*fakePaymentRepo)(nil)

func newFakePaymentRepo() *fakePaymentRepo {
	return &fakePaymentRepo{items: make(map[string]domain.Payment)}
}

func (f *fakePaymentRepo) Create(_ context.Context, payment domain.Payment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.items[payment.ID] = payment
	return nil
}

func (f *fakePaymentRepo) Get(_ context.Context, id string) (domain.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return domain.Payment{}, f.getErr
	}
	payment, ok := f.items[id]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}
	return payment, nil
}

func (f *fakePaymentRepo) ListByClinic(_ context.Context, clinicID string) ([]domain.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	list := make([]domain.Payment, 0)
	for _, payment := range f.items {
		if payment.ClinicID == clinicID {
			list = append(list, payment)
		}
	}
	return list, nil
}

func (f *fakePaymentRepo) Approve(_ context.Context, id string) (domain.Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.approveErr != nil {
		return domain.Payment{}, f.approveErr
	}
	payment, ok := f.items[id]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}
	if err := payment.Approve(); err != nil {
		return domain.Payment{}, err
	}
	f.items[id] = payment
	return payment, nil
}

func (f *fakePaymentRepo) get(id string) (domain.Payment, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	payment, ok := f.items[id]
	return payment, ok
}

type fakePixProvider struct {
	code  string
	err   error
	calls int
}

var _ port.PixProvider = (*fakePixProvider)(nil)

func (f *fakePixProvider) GenerateCharge(_ context.Context, _ port.PixChargeInput) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.code, nil
}

func seedClinic(t *testing.T, repo *fakeClinicRepo) domain.Clinic {
	t.Helper()
	doc, err := domain.NewDocument("11222333000181")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	clinic, err := domain.NewClinic(doc, "Clínica Sorriso LTDA", "Sorriso Odonto")
	if err != nil {
		t.Fatalf("NewClinic: %v", err)
	}
	repo.items[clinic.ID] = *clinic
	return *clinic
}

func seedDentist(t *testing.T, repo *fakeDentistRepo, clinicID string) domain.Dentist {
	t.Helper()
	dentist, err := domain.NewDentist(clinicID, "Dra. Ana Souza", "(11) 98765-4321", "ana@x.com", true)
	if err != nil {
		t.Fatalf("NewDentist: %v", err)
	}
	repo.items[dentist.ID] = *dentist
	return *dentist
}
