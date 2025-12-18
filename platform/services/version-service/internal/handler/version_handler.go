package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zerodayz7/platform/services/version-service/config"
	"github.com/zerodayz7/platform/services/version-service/internal/model"
)

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	resp := model.VersionResponse{
		MinVersion:  cfg.Min,
		Latest:      cfg.Latest,
		ForceUpdate: cfg.Force,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
