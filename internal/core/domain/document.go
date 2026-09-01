package domain

import "strings"

// DocumentType identifica o tipo de documento fiscal de uma clínica.
type DocumentType string

// Tipos de documento aceitos.
const (
	DocumentTypeCPF  DocumentType = "cpf"
	DocumentTypeCNPJ DocumentType = "cnpj"
)

const (
	lenCPF  = 11
	lenCNPJ = 14
	// cnpjBaseLen é a parte alfanumérica do CNPJ: 8 caracteres de raiz +
	// 4 de ordem do estabelecimento. Os 2 restantes são o DV, numérico.
	cnpjBaseLen = 12
)

// Document é o CPF ou CNPJ de uma clínica, sempre validado e normalizado
// (sem máscara, letras em maiúsculas) na construção — um Document não-zero
// é, por definição, um documento válido. A struct é comparável: dois
// documentos iguais com máscaras de entrada diferentes resultam em valores
// iguais.
type Document struct {
	value   string
	docType DocumentType
}

// NewDocument valida e normaliza um CPF (11 dígitos) ou CNPJ (14
// caracteres), aceitando entrada com ou sem máscara e com letras
// minúsculas.
//
// Para CNPJ é aceito o formato alfanumérico vigente desde julho/2026:
// letras A–Z e dígitos nas 12 primeiras posições (raiz e ordem), com os
// dois dígitos verificadores sempre numéricos. Além do tamanho e do
// conjunto de caracteres, valida os dígitos verificadores e rejeita
// sequências de um único caractere repetido, que passam no cálculo dos
// verificadores mas não são documentos reais.
func NewDocument(raw string) (Document, error) {
	clean := normalizeDocument(raw)
	if clean == "" {
		return Document{}, invalidInput("documento é obrigatório")
	}
	if allSameChars(clean) {
		return Document{}, invalidInput("documento inválido")
	}

	switch len(clean) {
	case lenCPF:
		if !allDigits(clean) {
			return Document{}, invalidInput("cpf deve conter apenas dígitos")
		}
		if !validCPF(clean) {
			return Document{}, invalidInput("dígitos verificadores do cpf não conferem")
		}
		return Document{value: clean, docType: DocumentTypeCPF}, nil
	case lenCNPJ:
		if !alphanumericUpper(clean[:cnpjBaseLen]) || !allDigits(clean[cnpjBaseLen:]) {
			return Document{}, invalidInput("cnpj deve conter dígitos ou letras A-Z, com verificadores numéricos")
		}
		if !validCNPJ(clean) {
			return Document{}, invalidInput("dígitos verificadores do cnpj não conferem")
		}
		return Document{value: clean, docType: DocumentTypeCNPJ}, nil
	default:
		return Document{}, invalidInput("documento deve ter 11 (cpf) ou 14 (cnpj) caracteres")
	}
}

// String retorna o documento normalizado, sem máscara.
func (d Document) String() string { return d.value }

// Type retorna o tipo do documento (cpf ou cnpj).
func (d Document) Type() DocumentType { return d.docType }

// IsZero indica um Document não inicializado.
func (d Document) IsZero() bool { return d.value == "" }

// Formatted retorna o documento com a máscara usual: 000.000.000-00 para
// CPF e 00.000.000/0000-00 para CNPJ (a mesma máscara vale para o CNPJ
// alfanumérico). Retorna string vazia para o Document zero.
func (d Document) Formatted() string {
	v := d.value
	switch d.docType {
	case DocumentTypeCPF:
		return v[:3] + "." + v[3:6] + "." + v[6:9] + "-" + v[9:]
	case DocumentTypeCNPJ:
		return v[:2] + "." + v[2:5] + "." + v[5:8] + "/" + v[8:12] + "-" + v[12:]
	default:
		return v
	}
}

// normalizeDocument remove máscara e espaços e converte letras para
// maiúsculas em uma única passada. Caracteres desconhecidos são
// preservados de propósito, para que a validação de conjunto de
// caracteres os rejeite depois.
func normalizeDocument(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c == '.' || c == '-' || c == '/' || c == ' ' || c == '\t':
		case c >= 'a' && c <= 'z':
			b.WriteByte(c - ('a' - 'A'))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func allSameChars(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func alphanumericUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

var (
	cpfWeights1  = []int{10, 9, 8, 7, 6, 5, 4, 3, 2}
	cpfWeights2  = []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}
	cnpjWeights1 = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	cnpjWeights2 = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

// checkDigit calcula um dígito verificador pelo módulo 11: soma ponderada
// dos caracteres, com resto menor que 2 resultando em 0.
//
// O valor de cada caractere é seu código ASCII menos 48, conforme a regra
// da Receita Federal para o CNPJ alfanumérico ('0'–'9' valem 0–9 e
// 'A'–'Z' valem 17–42) — para documentos puramente numéricos o resultado
// é idêntico ao cálculo tradicional.
func checkDigit(chars string, weights []int) byte {
	sum := 0
	for i, w := range weights {
		sum += (int(chars[i]) - '0') * w
	}
	if r := sum % 11; r >= 2 {
		return byte('0' + 11 - r)
	}
	return '0'
}

func validCPF(d string) bool {
	return d[9] == checkDigit(d, cpfWeights1) && d[10] == checkDigit(d, cpfWeights2)
}

func validCNPJ(d string) bool {
	return d[12] == checkDigit(d, cnpjWeights1) && d[13] == checkDigit(d, cnpjWeights2)
}
