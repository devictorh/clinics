package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devictorh/clinics/internal/adapter/httpapi"
	"github.com/devictorh/clinics/internal/adapter/httpapi/middleware"
	"github.com/devictorh/clinics/internal/adapter/memory"
	"github.com/devictorh/clinics/internal/adapter/pixsim"
	"github.com/devictorh/clinics/internal/core/service"
)

func newAPI(t *testing.T) http.Handler {
	t.Helper()
	clinicRepo := memory.NewClinicRepository()
	dentistRepo := memory.NewDentistRepository()
	paymentRepo := memory.NewPaymentRepository()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	paymentSvc := service.NewPaymentService(paymentRepo, clinicRepo, dentistRepo, pixsim.NewProvider(),
		service.WithApprovalDelay(func() time.Duration { return 5 * time.Millisecond }))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = paymentSvc.Shutdown(ctx)
	})

	return httpapi.NewRouter(
		service.NewClinicService(clinicRepo, dentistRepo),
		service.NewDentistService(dentistRepo, clinicRepo),
		paymentSvc,
		logger,
	)
}

func doJSON(t *testing.T, api http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode resposta %q: %v", rec.Body.String(), err)
	}
	return out
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, quer %d; corpo: %s", rec.Code, want, rec.Body.String())
	}
}

func wantErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	env := decodeBody[map[string]map[string]string](t, rec)
	if got := env["error"]["code"]; got != want {
		t.Errorf("error.code = %q, quer %q", got, want)
	}
}

type clinicBody struct {
	ID           string          `json:"id"`
	Document     string          `json:"document"`
	DocumentType string          `json:"document_type"`
	LegalName    string          `json:"legal_name"`
	TradeName    string          `json:"trade_name"`
	BankAccount  *map[string]any `json:"bank_account"`
}

type dentistBody struct {
	ID       string `json:"id"`
	ClinicID string `json:"clinic_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Admin    bool   `json:"admin"`
}

func createClinic(t *testing.T, api http.Handler, document string) clinicBody {
	t.Helper()
	rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics", map[string]any{
		"document": document, "legal_name": "Clínica Sorriso LTDA", "trade_name": "Sorriso Odonto",
	})
	wantStatus(t, rec, http.StatusCreated)
	return decodeBody[clinicBody](t, rec)
}

func TestClinicCRUDFlow(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	clinic := createClinic(t, api, "11.222.333/0001-81")
	if clinic.ID == "" || clinic.Document != "11222333000181" || clinic.DocumentType != "cnpj" {
		t.Fatalf("clínica criada = %+v", clinic)
	}
	if clinic.BankAccount != nil {
		t.Error("clínica sem dados bancários deveria ter bank_account null")
	}

	rec := doJSON(t, api, http.MethodGet, "/api/v1/clinics", nil)
	wantStatus(t, rec, http.StatusOK)
	if list := decodeBody[[]clinicBody](t, rec); len(list) != 1 {
		t.Errorf("lista = %d itens, quer 1", len(list))
	}

	rec = doJSON(t, api, http.MethodGet, "/api/v1/clinics/"+clinic.ID, nil)
	wantStatus(t, rec, http.StatusOK)

	rec = doJSON(t, api, http.MethodPut, "/api/v1/clinics/"+clinic.ID, map[string]any{
		"legal_name": "Nova Razão LTDA", "trade_name": "Novo Fantasia",
	})
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody[clinicBody](t, rec); got.LegalName != "Nova Razão LTDA" {
		t.Errorf("legal_name = %q", got.LegalName)
	}

	rec = doJSON(t, api, http.MethodPut, "/api/v1/clinics/"+clinic.ID+"/bank-account", map[string]any{
		"bank": "341", "agency": "0001", "account": "12345-6",
	})
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody[clinicBody](t, rec); got.BankAccount == nil {
		t.Error("bank_account deveria estar preenchido")
	}

	rec = doJSON(t, api, http.MethodDelete, "/api/v1/clinics/"+clinic.ID, nil)
	wantStatus(t, rec, http.StatusNoContent)

	rec = doJSON(t, api, http.MethodGet, "/api/v1/clinics/"+clinic.ID, nil)
	wantStatus(t, rec, http.StatusNotFound)
	wantErrorCode(t, rec, "not_found")
}

func TestClinicErrors(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	t.Run("json malformado", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics", bytes.NewReader([]byte("{invalido")))
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, req)
		wantStatus(t, rec, http.StatusBadRequest)
		wantErrorCode(t, rec, "invalid_input")
	})

	t.Run("documento inválido", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics", map[string]any{
			"document": "123", "legal_name": "Razão", "trade_name": "Fantasia",
		})
		wantStatus(t, rec, http.StatusBadRequest)
		wantErrorCode(t, rec, "invalid_input")
	})

	t.Run("documento duplicado", func(t *testing.T) {
		t.Parallel()
		createClinic(t, api, "529.982.247-25")
		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics", map[string]any{
			"document": "52998224725", "legal_name": "Outra Razão", "trade_name": "Outra",
		})
		wantStatus(t, rec, http.StatusConflict)
		wantErrorCode(t, rec, "document_already_exists")
	})

	t.Run("update de inexistente", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodPut, "/api/v1/clinics/nao-existe", map[string]any{
			"legal_name": "A", "trade_name": "B",
		})
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("rota desconhecida com envelope json", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodGet, "/api/v2/nada", nil)
		wantStatus(t, rec, http.StatusNotFound)
		wantErrorCode(t, rec, "not_found")
	})
}

func TestDentistFlow(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	clinic := createClinic(t, api, "11.222.333/0001-81")
	base := "/api/v1/clinics/" + clinic.ID + "/dentists"

	rec := doJSON(t, api, http.MethodPost, base, map[string]any{
		"name": "Dra. Ana Souza", "phone": "(11) 98765-4321", "email": "ana@sorriso.com.br", "admin": true,
	})
	wantStatus(t, rec, http.StatusCreated)
	dentist := decodeBody[dentistBody](t, rec)
	if dentist.ClinicID != clinic.ID || dentist.Phone != "11987654321" || !dentist.Admin {
		t.Fatalf("dentista criado = %+v", dentist)
	}

	rec = doJSON(t, api, http.MethodGet, base, nil)
	wantStatus(t, rec, http.StatusOK)
	if list := decodeBody[[]dentistBody](t, rec); len(list) != 1 {
		t.Errorf("lista = %d itens, quer 1", len(list))
	}

	rec = doJSON(t, api, http.MethodPut, base+"/"+dentist.ID, map[string]any{
		"name": "Dr. Bruno Lima", "phone": "11 3265-4321", "email": "bruno@sorriso.com.br", "admin": false,
	})
	wantStatus(t, rec, http.StatusOK)
	if got := decodeBody[dentistBody](t, rec); got.Name != "Dr. Bruno Lima" || got.Admin {
		t.Errorf("dentista atualizado = %+v", got)
	}

	rec = doJSON(t, api, http.MethodDelete, base+"/"+dentist.ID, nil)
	wantStatus(t, rec, http.StatusNoContent)

	rec = doJSON(t, api, http.MethodGet, base+"/"+dentist.ID, nil)
	wantStatus(t, rec, http.StatusNotFound)
}

func TestDentistErrors(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	t.Run("clínica inexistente", func(t *testing.T) {
		t.Parallel()
		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/nao-existe/dentists", map[string]any{
			"name": "Ana", "phone": "(11) 98765-4321", "email": "ana@x.com",
		})
		wantStatus(t, rec, http.StatusNotFound)
	})

	t.Run("email inválido", func(t *testing.T) {
		t.Parallel()
		clinic := createClinic(t, api, "11.222.333/0001-81")
		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/"+clinic.ID+"/dentists", map[string]any{
			"name": "Ana", "phone": "(11) 98765-4321", "email": "sem-arroba",
		})
		wantStatus(t, rec, http.StatusBadRequest)
		wantErrorCode(t, rec, "invalid_input")
	})

	t.Run("dentista de outra clínica responde not found", func(t *testing.T) {
		t.Parallel()
		clinicA := createClinic(t, api, "12.ABC.345/01DE-35")
		clinicB := createClinic(t, api, "529.982.247-25")

		rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/"+clinicA.ID+"/dentists", map[string]any{
			"name": "Ana", "phone": "(11) 98765-4321", "email": "ana@x.com",
		})
		wantStatus(t, rec, http.StatusCreated)
		dentist := decodeBody[dentistBody](t, rec)

		rec = doJSON(t, api, http.MethodGet, "/api/v1/clinics/"+clinicB.ID+"/dentists/"+dentist.ID, nil)
		wantStatus(t, rec, http.StatusNotFound)
	})
}

func TestClinicDeleteCascade(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	clinic := createClinic(t, api, "11.222.333/0001-81")
	rec := doJSON(t, api, http.MethodPost, "/api/v1/clinics/"+clinic.ID+"/dentists", map[string]any{
		"name": "Ana", "phone": "(11) 98765-4321", "email": "ana@x.com",
	})
	wantStatus(t, rec, http.StatusCreated)
	dentist := decodeBody[dentistBody](t, rec)

	rec = doJSON(t, api, http.MethodDelete, "/api/v1/clinics/"+clinic.ID, nil)
	wantStatus(t, rec, http.StatusNoContent)

	rec = doJSON(t, api, http.MethodGet, "/api/v1/clinics/"+clinic.ID+"/dentists/"+dentist.ID, nil)
	wantStatus(t, rec, http.StatusNotFound)
}

func TestRequestID(t *testing.T) {
	t.Parallel()
	api := newAPI(t)

	rec := doJSON(t, api, http.MethodGet, "/healthz", nil)
	wantStatus(t, rec, http.StatusOK)
	if rec.Header().Get(middleware.HeaderRequestID) == "" {
		t.Error("resposta sem X-Request-ID gerado")
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(middleware.HeaderRequestID, "meu-id-123")
	echo := httptest.NewRecorder()
	api.ServeHTTP(echo, req)
	if got := echo.Header().Get(middleware.HeaderRequestID); got != "meu-id-123" {
		t.Errorf("X-Request-ID = %q, quer eco de %q", got, "meu-id-123")
	}
}
