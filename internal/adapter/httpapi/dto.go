package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/devictorh/clinics/internal/core/domain"
	"github.com/devictorh/clinics/internal/core/service"
)

const maxBodyBytes = 1 << 20

type bankAccountPayload struct {
	Bank    string `json:"bank"`
	Agency  string `json:"agency"`
	Account string `json:"account"`
}

type createClinicRequest struct {
	Document    string              `json:"document"`
	LegalName   string              `json:"legal_name"`
	TradeName   string              `json:"trade_name"`
	BankAccount *bankAccountPayload `json:"bank_account"`
}

func (r createClinicRequest) toInput() service.CreateClinicInput {
	in := service.CreateClinicInput{
		Document:  r.Document,
		LegalName: r.LegalName,
		TradeName: r.TradeName,
	}
	if r.BankAccount != nil {
		in.BankAccount = &service.BankAccountInput{
			Bank:    r.BankAccount.Bank,
			Agency:  r.BankAccount.Agency,
			Account: r.BankAccount.Account,
		}
	}
	return in
}

type updateClinicRequest struct {
	LegalName string `json:"legal_name"`
	TradeName string `json:"trade_name"`
}

type clinicResponse struct {
	ID           string              `json:"id"`
	Document     string              `json:"document"`
	DocumentType string              `json:"document_type"`
	LegalName    string              `json:"legal_name"`
	TradeName    string              `json:"trade_name"`
	BankAccount  *bankAccountPayload `json:"bank_account"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

func toClinicResponse(c domain.Clinic) clinicResponse {
	resp := clinicResponse{
		ID:           c.ID,
		Document:     c.Document.String(),
		DocumentType: string(c.Document.Type()),
		LegalName:    c.LegalName,
		TradeName:    c.TradeName,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
	if !c.BankAccount.IsZero() {
		resp.BankAccount = &bankAccountPayload{
			Bank:    c.BankAccount.Bank,
			Agency:  c.BankAccount.Agency,
			Account: c.BankAccount.Account,
		}
	}
	return resp
}

func toClinicResponses(clinics []domain.Clinic) []clinicResponse {
	list := make([]clinicResponse, len(clinics))
	for i, c := range clinics {
		list[i] = toClinicResponse(c)
	}
	return list
}

type dentistRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}

func (r dentistRequest) toInput() service.DentistInput {
	return service.DentistInput{Name: r.Name, Phone: r.Phone, Email: r.Email, Admin: r.Admin}
}

type dentistResponse struct {
	ID        string    `json:"id"`
	ClinicID  string    `json:"clinic_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toDentistResponse(d domain.Dentist) dentistResponse {
	return dentistResponse{
		ID:        d.ID,
		ClinicID:  d.ClinicID,
		Name:      d.Name,
		Phone:     d.Phone,
		Email:     d.Email,
		Admin:     d.Admin,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func toDentistResponses(dentists []domain.Dentist) []dentistResponse {
	list := make([]dentistResponse, len(dentists))
	for i, d := range dentists {
		list[i] = toDentistResponse(d)
	}
	return list
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("%w: corpo da requisição inválido", domain.ErrInvalidInput)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
