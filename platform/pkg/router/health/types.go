package health

import "time"

type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Version string            `json:"version"`
	Time    string            `json:"time"`
	Checks  map[string]string `json:"checks"`
}

// NewResponse to pomocnik do tworzenia odpowiedzi
func NewResponse(service, version string, checks map[string]string) HealthResponse {
	status := "ok"
	for _, v := range checks {
		if v != "ok" {
			status = "degraded"
			break
		}
	}

	return HealthResponse{
		Status:  status,
		Service: service,
		Version: version,
		Time:    time.Now().UTC().Format(time.RFC3339),
		Checks:  checks,
	}
}
