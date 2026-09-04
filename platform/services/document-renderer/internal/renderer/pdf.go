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
	PaperWidthInches   float64 `json:"paper_width_inches"`
	PaperHeightInches  float64 `json:"paper_height_inches"`
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

type PDFRenderer interface {
	RenderHTMLToPDF(ctx context.Context, htmlContent string, opts PDFOptions) ([]byte, error)
}

type RodPDFRenderer struct {
	browser *rod.Browser
}

func NewRodPDFRenderer(browser *rod.Browser) PDFRenderer {
	return &RodPDFRenderer{
		browser: browser,
	}
}

func (r *RodPDFRenderer) RenderHTMLToPDF(ctx context.Context, htmlContent string, opts PDFOptions) ([]byte, error) {
	incognitoCtx, err := r.browser.Incognito()
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
