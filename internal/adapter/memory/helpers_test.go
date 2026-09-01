package memory_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
)

// cnpjFor gera o n-ésimo CNPJ numérico válido, calculando os dígitos
// verificadores — os testes de concorrência precisam de muitos documentos
// distintos.
func cnpjFor(t *testing.T, n int) string {
	t.Helper()
	base := fmt.Sprintf("%012d", n)
	base += checkDigit(base, []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	base += checkDigit(base, []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	return base
}

func checkDigit(chars string, weights []int) string {
	sum := 0
	for i, w := range weights {
		sum += (int(chars[i]) - '0') * w
	}
	if r := sum % 11; r >= 2 {
		return string(byte('0' + 11 - r))
	}
	return "0"
}

func newClinic(t *testing.T, rawDoc string) domain.Clinic {
	t.Helper()
	doc, err := domain.NewDocument(rawDoc)
	if err != nil {
		t.Fatalf("NewDocument(%q): %v", rawDoc, err)
	}
	c, err := domain.NewClinic(doc, "Clínica "+rawDoc+" LTDA", "Fantasia "+rawDoc)
	if err != nil {
		t.Fatalf("NewClinic: %v", err)
	}
	return *c
}

func newDentist(t *testing.T, clinicID, name string) domain.Dentist {
	t.Helper()
	return newDentistWithEmail(t, clinicID, name, emailFor(name))
}

func newDentistWithEmail(t *testing.T, clinicID, name, email string) domain.Dentist {
	t.Helper()
	d, err := domain.NewDentist(clinicID, name, "(11) 98765-4321", email, false)
	if err != nil {
		t.Fatalf("NewDentist: %v", err)
	}
	return *d
}

// emailFor deriva um email ASCII do nome, para que dentistas de teste com
// nomes distintos não colidam no índice de unicidade.
func emailFor(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String() + "@x.com"
}

func at(base time.Time, offset time.Duration) time.Time {
	return base.Add(offset)
}
