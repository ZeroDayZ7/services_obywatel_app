package renderer

import (
	"context"
	"fmt"
	"io"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type PDFOptions struct {
	Landscape          bool    `json:"landscape"`
	PrintBackground    bool    `json:"print_background"`
	MarginTopInches    float64 `json:"margin_top_inches"`
	MarginBottomInches float64 `json:"margin_bottom_inches"`
	MarginLeftInches   float64 `json:"margin_left_inches"`
	MarginRightInches  float64 `json:"margin_right_inches"`
	PaperWidthInches   float64 `json:"paper_width_inches"`  // opcjonalnie
	PaperHeightInches  float64 `json:"paper_height_inches"` // opcjonalnie
}

func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		Landscape:          false,
		PrintBackground:    true,
		MarginTopInches:    0.4,
		MarginBottomInches: 0.4,
		MarginLeftInches:   0.4,
		MarginRightInches:  0.4,
	}
}

func RenderHTMLToPDF(ctx context.Context, browser *rod.Browser, htmlContent string, opts PDFOptions) ([]byte, error) {
	// Tworzymy wyizolowany IncognitoContext dla bezpieczeństwa session storage/cookies
	incognitoCtx, err := browser.Incognito()
	if err != nil {
		return nil, fmt.Errorf("failed to create incognito context: %w", err)
	}
	defer incognitoCtx.Close()

	page, err := incognitoCtx.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	pageWithCtx := page.Context(ctx)

	if err := pageWithCtx.SetDocumentContent(htmlContent); err != nil {
		return nil, fmt.Errorf("failed to set content: %w", err)
	}

	// Czekamy aż strona i ewentualne zasoby się ustabilizują
	_ = pageWithCtx.WaitStable(200)

	pdfRequest := &proto.PagePrintToPDF{
		Landscape:         opts.Landscape,
		PrintBackground:   opts.PrintBackground,
		MarginTop:         &opts.MarginTopInches,
		MarginBottom:      &opts.MarginBottomInches,
		MarginLeft:        &opts.MarginLeftInches,
		MarginRight:       &opts.MarginRightInches,
		PreferCSSPageSize: true,
	}

	if opts.PaperWidthInches > 0 {
		pdfRequest.PaperWidth = &opts.PaperWidthInches
	}
	if opts.PaperHeightInches > 0 {
		pdfRequest.PaperHeight = &opts.PaperHeightInches
	}

	res, err := pageWithCtx.PDF(pdfRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to print pdf: %w", err)
	}

	buf, err := io.ReadAll(res)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdf stream: %w", err)
	}

	return buf, nil
}
