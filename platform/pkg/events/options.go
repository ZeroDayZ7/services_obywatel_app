package events

// EmitOption
type EmitOption func(*Event)

func WithFlags(flags EventFlags) EmitOption {
	return func(e *Event) {
		e.Flags = flags
	}
}

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

func WithIP(ip string) EmitOption {
	return func(e *Event) {
		e.IP = ip
	}
}

func WithVersion(v int) EmitOption {
	return func(e *Event) {
		e.Version = v
	}
}
