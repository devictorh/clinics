package domain

import (
	"errors"
	"fmt"
)

// Erros sentinela do domínio. Os adapters os traduzem para as respostas
// adequadas (ex.: HTTP 404/409/400) via errors.Is.
var (
	// ErrNotFound indica que o registro não existe ou foi excluído (soft delete).
	ErrNotFound = errors.New("registro não encontrado")

	// ErrDocumentAlreadyExists indica que já existe uma clínica ativa com o
	// mesmo documento.
	ErrDocumentAlreadyExists = errors.New("documento já cadastrado")

	// ErrEmailAlreadyExists indica que já existe um dentista ativo com o
	// mesmo email na clínica.
	ErrEmailAlreadyExists = errors.New("email já cadastrado na clínica")

	// ErrInvalidInput indica dados de entrada que violam invariantes do
	// domínio. Erros de validação embrulham este sentinela e são
	// detectáveis com errors.Is.
	ErrInvalidInput = errors.New("dados inválidos")

	// ErrInvalidStatusTransition indica uma transição de status de
	// pagamento fora do ciclo pending → approved.
	ErrInvalidStatusTransition = errors.New("transição de status inválida")
)

func invalidInput(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, msg)
}
