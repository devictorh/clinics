package middleware

import (
	"context"
	"net/http"
	"uuid"
)

// HeaderRequestID é o header de correlação aceito e devolvido pela API.
const HeaderRequestID = "X-Request-ID"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
)

// RequestID propaga o request ID do header para o contexto e para a
// resposta, gerando um UUID quando o cliente não envia o seu.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set(HeaderRequestID, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom retorna o request ID do contexto, ou vazio.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
