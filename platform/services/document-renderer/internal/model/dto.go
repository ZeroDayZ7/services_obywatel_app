package model

type PDFOptions struct {
	Landscape          *bool    `json:"landscape,omitempty"`
	PrintBackground    *bool    `json:"print_background,omitempty"`
	MarginTopInches    *float64 `json:"margin_top_inches,omitempty"`
	MarginBottomInches *float64 `json:"margin_bottom_inches,omitempty"`
	MarginLeftInches   *float64 `json:"margin_left_inches,omitempty"`
	MarginRightInches  *float64 `json:"margin_right_inches,omitempty"`
	PaperWidthInches   *float64 `json:"paper_width_inches,omitempty"`
	PaperHeightInches  *float64 `json:"paper_height_inches,omitempty"`
}

type RenderDocumentRequest struct {
	Template string         `json:"template"`
	Data     map[string]any `json:"data"`
	Options  *PDFOptions    `json:"options,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
