package events

// EventFlags
type EventFlags struct {
	Audit  bool `json:"audit"`
	Notify bool `json:"notify"`
	Store  bool `json:"store"`
}
