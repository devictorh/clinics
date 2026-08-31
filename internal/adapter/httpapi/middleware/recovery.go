package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery converte panics em 500 com envelope JSON, registrando o stack
// trace — um handler com bug não pode derrubar o servidor.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				LoggerFrom(r.Context()).Error("panic recuperado",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"erro interno do servidor"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
