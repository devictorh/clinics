package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/devictorh/clinics/internal/adapter/httpapi/middleware"
	"github.com/devictorh/clinics/internal/core/domain"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError traduz erros de domínio para o envelope JSON e o status HTTP
// correspondentes; erros desconhecidos viram 500 sem vazar detalhes.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorEnvelope{errorBody{
			Code: "not_found", Message: domain.ErrNotFound.Error(),
		}})
	case errors.Is(err, domain.ErrDocumentAlreadyExists):
		writeJSON(w, http.StatusConflict, errorEnvelope{errorBody{
			Code: "document_already_exists", Message: domain.ErrDocumentAlreadyExists.Error(),
		}})
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		writeJSON(w, http.StatusConflict, errorEnvelope{errorBody{
			Code: "email_already_exists", Message: domain.ErrEmailAlreadyExists.Error(),
		}})
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorEnvelope{errorBody{
			Code: "invalid_input", Message: err.Error(),
		}})
	default:
		middleware.LoggerFrom(r.Context()).Error("erro não mapeado", slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorEnvelope{errorBody{
			Code: "internal_error", Message: "erro interno do servidor",
		}})
	}
}
