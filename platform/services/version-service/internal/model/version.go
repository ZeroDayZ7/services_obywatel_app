package model

type VersionResponse struct {
	MinVersion  string `json:"minVersion"`
	Latest      string `json:"latestVersion"`
	ForceUpdate bool   `json:"forceUpdate"`
}
