package validator

// ResetSendResponse zwraca token sesji po wysłaniu maila (Krok 1)
type ResetSendResponse struct {
	Success    bool   `json:"success"`
	ResetToken string `json:"reset_token"`
}

// ResetVerifyResponse zwraca wyzwanie (challenge) po wpisaniu kodu OTP
type ResetVerifyResponse struct {
	Success    bool   `json:"success"`
	ResetToken string `json:"reset_token"`
	UserID     string `json:"user_id"`
	Challenge  string `json:"challenge"`
}

// ResetFinalResponse zwraca proste potwierdzenie sukcesu
type ResetFinalResponse struct {
	Success bool `json:"success"`
}
