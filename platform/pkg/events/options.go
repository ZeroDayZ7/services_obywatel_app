package events

// EmitOption
type EmitOption func(*Event)

//#region WithFlags
func WithFlags(flags EventFlags) EmitOption {
	return func(e *Event) {
		e.Flags = flags
	}
}

//#region WithMetadata
func WithMetadata(metadata map[string]any) EmitOption {
	return func(e *Event) {
		if e.Metadata == nil {
			e.Metadata = map[string]any{}
		}
		for k, v := range metadata {
			e.Metadata[k] = v
		}
	}
}

//#region WithIP
func WithIP(ip string) EmitOption {
	return func(e *Event) {
		e.IP = ip
	}
}

//#region WithVersion
func WithVersion(v int) EmitOption {
	return func(e *Event) {
		e.Version = v
	}
}
