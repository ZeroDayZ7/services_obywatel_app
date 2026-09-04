package renderer

import (
	"context"
	"document-renderer/internal/model"
	"fmt"
	"io"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func DefaultPDFOptions() model.PDFOptions {
	return model.PDFOptions{
		Landscape:          false,
		PrintBackground:    true,
		MarginTopInches:    0.4,
		MarginBottomInches: 0.4,
		MarginLeftInches:   0.4,
		MarginRightInches:  0.4,
	}
}

type PDFRenderer interface {
	RenderHTMLToPDF(ctx context.Context, htmlContent string, opts model.PDFOptions) ([]byte, error)
	Ping(ctx context.Context) error
}

type RodPDFRenderer struct {
	browser       *rod.Browser
	semaphore     chan struct{}
	renderTimeout time.Duration
}

func NewRodPDFRenderer(browser *rod.Browser, maxConcurrency int, renderTimeout time.Duration) PDFRenderer {
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	return &RodPDFRenderer{
		browser:       browser,
		semaphore:     make(chan struct{}, maxConcurrency),
		renderTimeout: renderTimeout,
	}
}

func (r *RodPDFRenderer) Ping(ctx context.Context) error {
	page, err := r.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return fmt.Errorf("chromium check failed: %w", err)
	}
	defer page.Close()
	return nil
}

func (r *RodPDFRenderer) RenderHTMLToPDF(ctx context.Context, htmlContent string, opts model.PDFOptions) ([]byte, error) {
	// Rezerwacja slotu w semaforze
	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	renderCtx, cancel := context.WithTimeout(ctx, r.renderTimeout)
	defer cancel()

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

	pageWithCtx := page.Context(renderCtx)

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
