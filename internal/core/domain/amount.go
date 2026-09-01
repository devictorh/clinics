package domain

import "fmt"

// Amount é um valor monetário em centavos de real (BRL). Inteiros evitam
// os erros de arredondamento de ponto flutuante em operações financeiras;
// a conversão para decimal acontece apenas na borda (apresentação).
type Amount int64

// NewAmount cria um Amount a partir de centavos. Cobranças exigem valor
// positivo.
func NewAmount(cents int64) (Amount, error) {
	if cents <= 0 {
		return 0, invalidInput("valor deve ser maior que zero")
	}
	return Amount(cents), nil
}

// Cents retorna o valor em centavos.
func (a Amount) Cents() int64 { return int64(a) }

// String formata o valor em reais com duas casas decimais (ex.: "150.00").
func (a Amount) String() string {
	return fmt.Sprintf("%d.%02d", a/100, a%100)
}
