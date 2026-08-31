package httpapi_test

import (
	"net/http"
	"testing"
	"time"
)

type paymentBody struct {
	ID               string  `json:"id"`
	ClinicID         string  `json:"clinic_id"`
	DentistID        *string `json:"dentist_id"`
	Amount           int64   `json:"amount"`
	Status           string  `json:"status"`
	PixCopyPasteCode string  `json:"pix_copy_paste_code"`
	ApprovedAt       *string `json:"approved_at"`
}

func TestPaymentFlow(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	clinic := createClinic(t, api, "11.222.333/0001-81")

	rec := doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
		"clinic_id": clinic.ID, "amount": 15000,
	})
	wantStatus(t, rec, http.StatusCreated)
	payment := decodeBody[paymentBody](t, rec)
	if payment.Status != "pending" || payment.PixCopyPasteCode == "" || payment.ApprovedAt != nil {
		t.Fatalf("cobrança criada = %+v", payment)
	}
	if payment.DentistID != nil {
		t.Error("dentist_id deveria ser null quando não informado")
	}
	if payment.Amount != 15000 {
		t.Errorf("amount = %d, quer 15000", payment.Amount)
	}

	// a confirmação simulada roda em background; acompanha via GET até aprovar
	deadline := time.Now().Add(2 * time.Second)
	var current paymentBody
	for {
		rec = doJSON(t, api, http.MethodGet, "/api/v1/payments/"+payment.ID, nil)
		wantStatus(t, rec, http.StatusOK)
		current = decodeBody[paymentBody](t, rec)
		if current.Status == "approved" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != "approved" || current.ApprovedAt == nil {
		t.Fatalf("cobrança não aprovada dentro do prazo: %+v", current)
	}

	rec = doJSON(t, api, http.MethodGet, "/api/v1/clinics/"+clinic.ID+"/payments", nil)
	wantStatus(t, rec, http.StatusOK)
	if list := decodeBody[[]paymentBody](t, rec); len(list) != 1 || list[0].ID != payment.ID {
		t.Errorf("histórico = %d itens", len(list))
	}
}

func TestPaymentComDentista(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	clinic := createClinic(t, api, "11.222.333/0001-81")
	rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/"+clinic.ID+"/dentists", map[string]any{
		"name": "Dra. Ana", "phone": "(11) 98765-4321", "email": "ana@x.com",
	})
	wantStatus(t, rec, http.StatusCreated)
	dentist := decodeBody[dentistBody](t, rec)

	rec = doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
		"clinic_id": clinic.ID, "amount": 5000, "dentist_id": dentist.ID,
	})
	wantStatus(t, rec, http.StatusCreated)
	payment := decodeBody[paymentBody](t, rec)
	if payment.DentistID == nil || *payment.DentistID != dentist.ID {
		t.Errorf("dentist_id = %v, quer %s", payment.DentistID, dentist.ID)
	}
}

func TestPaymentErrors(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	t.Run("valor inválido", func(t *testing.T) {
		t.Parallel()
		clinic := createClinic(t, api, "529.982.247-25")
		rec := doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
			"clinic_id": clinic.ID, "amount": 0,
		})
		wantStatus(t, rec, http.StatusBadRequest)
		wantErrorCode(t, rec, "invalid_input")
	})

	t.Run("clínica inexistente", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
			"clinic_id": "nao-existe", "amount": 100,
		})
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("clínica excluída não recebe cobrança", func(t *testing.T) {
		t.Parallel()
		clinic := createClinic(t, api, "11.222.333/0001-81")
		rec := doJSON(t, api, http.MethodDelete, "/api/v1/clinics/"+clinic.ID, nil)
		wantStatus(t, rec, http.StatusNoContent)

		rec = doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
			"clinic_id": clinic.ID, "amount": 100,
		})
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("dentista de outra clínica", func(t *testing.T) {
		t.Parallel()
		clinicA := createClinic(t, api, "12.ABC.345/01DE-35")
		clinicB := createClinic(t, api, "111.444.777-35")
		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/"+clinicA.ID+"/dentists", map[string]any{
			"name": "Ana", "phone": "(11) 98765-4321", "email": "ana@x.com",
		})
		wantStatus(t, rec, http.StatusCreated)
		dentist := decodeBody[dentistBody](t, rec)

		rec = doJSON(t, api, http.MethodPost, "/api/v1/payments", map[string]any{
			"clinic_id": clinicB.ID, "amount": 100, "dentist_id": dentist.ID,
		})
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("cobrança inexistente", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodGet, "/api/v1/payments/nao-existe", nil)
		wantStatus(t, rec, http.StatusNotFound)
	})
}
