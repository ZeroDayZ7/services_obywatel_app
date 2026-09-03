package service

import (
	"context"
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type PDFGenerator interface {
	GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error)
}

type pdfGenerator struct{}

func NewPDFGenerator() PDFGenerator {
	return &pdfGenerator{}
}

type AgreementTemplateData struct {
	AgreementID     string
	AgreementNumber string
	FirstName       string
	SecondName      *string
	LastName        string
	PESEL           string
	Email           string
	PhoneNumber     string
	Street          string
	HouseNumber     string
	FlatNumber      *string
	PostalCode      string
	City            string
	SignedAt        string
	KeyVersion      int
	DocumentHash    string

	OfficerName   string
	OfficerID     string
	DepartmentID  string
	InstitutionID string
}

func (p *pdfGenerator) GenerateAgreementPDF(ctx context.Context, data AgreementTemplateData) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithOrientation(orientation.Vertical).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()

	m := maroto.New(cfg)

	// --- NAGŁÓWEK DOKUMENTU ---
	m.AddRows(
		row.New(12).Add(
			text.NewCol(12, "POTWIERDZENIE ZŁOŻENIA OŚWIADCZENIA", props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Align: align.Center,
				Color: &props.Color{Red: 30, Green: 64, Blue: 175},
			}),
		),
		row.New(8).Add(
			text.NewCol(12, fmt.Sprintf("Numer umowy: %s", data.AgreementNumber), props.Text{
				Size:  10,
				Style: fontstyle.Bold,
				Align: align.Center,
				Color: &props.Color{Red: 100, Green: 116, Blue: 139},
			}),
		),
		line.NewRow(2, props.Line{
			Color: &props.Color{Red: 226, Green: 232, Blue: 240},
		}),
	)

	// --- DANE OBYWATELA ---
	m.AddRows(
		row.New(8),
		row.New(8).Add(
			text.NewCol(12, "DANE WNIOSKODAWCY", props.Text{
				Size:  11,
				Style: fontstyle.Bold,
				Color: &props.Color{Red: 15, Green: 23, Blue: 42},
			}),
		),
		row.New(6).Add(
			text.NewCol(6, fmt.Sprintf("Imię i nazwisko: %s", formatFullName(data.FirstName, data.SecondName, data.LastName)), props.Text{Size: 9}),
			text.NewCol(6, fmt.Sprintf("PESEL: %s", data.PESEL), props.Text{Size: 9}),
		),
		row.New(6).Add(
			text.NewCol(6, fmt.Sprintf("E-mail: %s", data.Email), props.Text{Size: 9}),
			text.NewCol(6, fmt.Sprintf("Telefon: %s", data.PhoneNumber), props.Text{Size: 9}),
		),
		row.New(6).Add(
			text.NewCol(12, fmt.Sprintf("Adres: %s", formatAddress(data.Street, data.HouseNumber, data.FlatNumber, data.PostalCode, data.City)), props.Text{Size: 9}),
		),
	)

	// --- DANE ORGANU REJESTRUJĄCEGO (JEŚLI OBECNE) ---
	if data.OfficerName != "" || data.InstitutionID != "" {
		m.AddRows(
			row.New(8),
			row.New(8).Add(
				text.NewCol(12, "DANE ORGANU REJESTRUJĄCEGO", props.Text{
					Size:  11,
					Style: fontstyle.Bold,
					Color: &props.Color{Red: 15, Green: 23, Blue: 42},
				}),
			),
			row.New(6).Add(
				text.NewCol(6, fmt.Sprintf("Urzędnik: %s (ID: %s)", data.OfficerName, data.OfficerID), props.Text{Size: 9}),
				text.NewCol(6, fmt.Sprintf("Instytucja: %s / Dept: %s", data.InstitutionID, data.DepartmentID), props.Text{Size: 9}),
			),
		)
	}

	// --- FOOTER Z KODEM QR I HASH ---
	m.AddRows(
		row.New(12),
		line.NewRow(1, props.Line{
			Color: &props.Color{Red: 226, Green: 232, Blue: 240},
		}),
		row.New(6),
		row.New(30).Add(
			col.New(8).Add(
				text.New(fmt.Sprintf("Podpisano dnia: %s", data.SignedAt), props.Text{Size: 8, Style: fontstyle.Bold}),
				text.New(fmt.Sprintf("Identyfikator umowy: %s", data.AgreementID), props.Text{Size: 8}),
				text.New(fmt.Sprintf("Wersja klucza KMS: v%d", data.KeyVersion), props.Text{Size: 8}),
				text.New(fmt.Sprintf("Hash dokumentu (SHA-256): %s", data.DocumentHash), props.Text{Size: 7, Color: &props.Color{Red: 100, Green: 116, Blue: 139}}),
			),
			code.NewQrCol(4, data.DocumentHash, props.Rect{
				Percent: 100,
				Center:  true,
			}),
		),
	)

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf_gen: failed to generate pdf: %w", err)
	}

	return document.GetBytes(), nil
}

func formatFullName(first string, second *string, last string) string {
	if second != nil && *second != "" {
		return fmt.Sprintf("%s %s %s", first, *second, last)
	}
	return fmt.Sprintf("%s %s", first, last)
}

func formatAddress(street, house string, flat *string, code, city string) string {
	if flat != nil && *flat != "" {
		return fmt.Sprintf("ul. %s %s/%s, %s %s", street, house, *flat, code, city)
	}
	return fmt.Sprintf("ul. %s %s, %s %s", street, house, code, city)
}
