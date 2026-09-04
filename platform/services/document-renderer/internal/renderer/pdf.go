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

type ResolvedPDFOptions struct {
	Landscape          bool
	PrintBackground    bool
	MarginTopInches    float64
	MarginBottomInches float64
	MarginLeftInches   float64
	MarginRightInches  float64
	PaperWidthInches   float64
	PaperHeightInches  float64
}

func ResolvePDFOptions(opts *model.PDFOptions) ResolvedPDFOptions {
	res := ResolvedPDFOptions{
		Landscape:          false,
		PrintBackground:    true,
		MarginTopInches:    0.4,
		MarginBottomInches: 0.4,
		MarginLeftInches:   0.4,
		MarginRightInches:  0.4,
	}

	if opts == nil {
		return res
	}

	if opts.Landscape != nil {
		res.Landscape = *opts.Landscape
	}
	if opts.PrintBackground != nil {
		res.PrintBackground = *opts.PrintBackground
	}
	if opts.MarginTopInches != nil {
		res.MarginTopInches = *opts.MarginTopInches
	}
	if opts.MarginBottomInches != nil {
		res.MarginBottomInches = *opts.MarginBottomInches
	}
	if opts.MarginLeftInches != nil {
		res.MarginLeftInches = *opts.MarginLeftInches
	}
	if opts.MarginRightInches != nil {
		res.MarginRightInches = *opts.MarginRightInches
	}
	if opts.PaperWidthInches != nil {
		res.PaperWidthInches = *opts.PaperWidthInches
	}
	if opts.PaperHeightInches != nil {
		res.PaperHeightInches = *opts.PaperHeightInches
	}

	return res
}

type PDFRenderer interface {
	RenderHTMLToPDF(ctx context.Context, htmlContent string, opts *model.PDFOptions) ([]byte, error)
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

func (r *RodPDFRenderer) RenderHTMLToPDF(ctx context.Context, htmlContent string, opts *model.PDFOptions) ([]byte, error) {
	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	resolvedOpts := ResolvePDFOptions(opts)

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
		Landscape:         resolvedOpts.Landscape,
		PrintBackground:   resolvedOpts.PrintBackground,
		MarginTop:         &resolvedOpts.MarginTopInches,
		MarginBottom:      &resolvedOpts.MarginBottomInches,
		MarginLeft:        &resolvedOpts.MarginLeftInches,
		MarginRight:       &resolvedOpts.MarginRightInches,
		PreferCSSPageSize: true,
	}

	if resolvedOpts.PaperWidthInches > 0 {
		pdfRequest.PaperWidth = &resolvedOpts.PaperWidthInches
	}
	if resolvedOpts.PaperHeightInches > 0 {
		pdfRequest.PaperHeight = &resolvedOpts.PaperHeightInches
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
