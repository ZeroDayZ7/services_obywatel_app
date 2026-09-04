package model

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

type RenderDocumentRequest struct {
	Template string         `json:"template"`
	Data     map[string]any `json:"data"`
	Options  *PDFOptions    `json:"options,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
