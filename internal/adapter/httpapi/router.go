package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/devictorh/clinics/internal/adapter/httpapi/middleware"
)

// NewRouter monta as rotas da API sobre o http.ServeMux nativo (métodos e
// path params nos patterns, Go 1.22+) e aplica a cadeia de middlewares:
// request ID → access log → recovery.
func NewRouter(clinics ClinicService, dentists DentistService, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /openapi.yaml", handleOpenAPISpec)
	mux.HandleFunc("GET /docs", handleSwaggerUI)

	ch := &clinicHandler{svc: clinics}
	mux.HandleFunc("POST /api/v1/clinics", ch.create)
	mux.HandleFunc("GET /api/v1/clinics", ch.list)
	mux.HandleFunc("GET /api/v1/clinics/{clinicID}", ch.get)
	mux.HandleFunc("PUT /api/v1/clinics/{clinicID}", ch.update)
	mux.HandleFunc("PUT /api/v1/clinics/{clinicID}/bank-account", ch.updateBankAccount)
	mux.HandleFunc("DELETE /api/v1/clinics/{clinicID}", ch.delete)

	dh := &dentistHandler{svc: dentists}
	mux.HandleFunc("POST /api/v1/clinics/{clinicID}/dentists", dh.create)
	mux.HandleFunc("GET /api/v1/clinics/{clinicID}/dentists", dh.list)
	mux.HandleFunc("GET /api/v1/clinics/{clinicID}/dentists/{dentistID}", dh.get)
	mux.HandleFunc("PUT /api/v1/clinics/{clinicID}/dentists/{dentistID}", dh.update)
	mux.HandleFunc("DELETE /api/v1/clinics/{clinicID}/dentists/{dentistID}", dh.delete)

	// rotas desconhecidas respondem com o mesmo envelope JSON dos erros
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, errorEnvelope{errorBody{
			Code: "not_found", Message: "rota não encontrada",
		}})
	})

	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)
	return handler
}
