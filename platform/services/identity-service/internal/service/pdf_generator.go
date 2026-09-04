// cmdr: internal\service\pdf_generator.go

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PDFGenerator interface {
	GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error)
}

type documentRendererClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPDFGenerator(documentRendererURL string) PDFGenerator {
	return &documentRendererClient{
		baseURL: documentRendererURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type AgreementTemplateData struct {
	AgreementID     string  `json:"agreement_id"`
	AgreementNumber string  `json:"agreement_number"`
	FirstName       string  `json:"first_name"`
	SecondName      *string `json:"second_name,omitempty"`
	LastName        string  `json:"last_name"`
	PESEL           string  `json:"pesel"`
	Email           string  `json:"email"`
	PhoneNumber     string  `json:"phone_number"`
	Street          string  `json:"street"`
	HouseNumber     string  `json:"house_number"`
	FlatNumber      *string `json:"flat_number,omitempty"`
	PostalCode      string  `json:"postal_code"`
	City            string  `json:"city"`
	SignedAt        string  `json:"signed_at"`
	KeyVersion      int     `json:"key_version"`
	DocumentHash    string  `json:"document_hash"`

	OfficerName   string `json:"officer_name,omitempty"`
	OfficerID     string `json:"officer_id,omitempty"`
	DepartmentID  string `json:"department_id,omitempty"`
	InstitutionID string `json:"institution_id,omitempty"`
}

// Request wysyłany do mikroserwisu document-renderer
type renderPDFRequest struct {
	Template string                `json:"template"`
	Data     AgreementTemplateData `json:"data"`
}

func (c *documentRendererClient) GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error) {
	reqBody := renderPDFRequest{
		Template: "contracts/identity_certificate.html",
		Data:     data,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/render", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: document-renderer service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pdf_gen: renderer returned status %d", resp.StatusCode)
	}

	// Odbieramy czysty bufor bajtów pliku PDF
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to read response body: %w", err)
	}

	return buf.Bytes(), nil
}
