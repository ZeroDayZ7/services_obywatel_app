package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type StreamPublisher interface {
	Publish(ctx context.Context, stream string, payload any) error
}

const (
	DefaultStream  = "events.core"
	DefaultVersion = 1
)

type Emitter struct {
	publisher StreamPublisher
	source    string
	stream    string
}

func NewEmitter(publisher StreamPublisher, source string) *Emitter {
	return &Emitter{
		publisher: publisher,
		source:    source,
		stream:    DefaultStream,
	}
}

func (e *Emitter) Emit(
	ctx context.Context,
	eventType EventType,
	userID string,
	opts ...EmitOption,
) error {

	evt := Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		UserID:    userID,
		Source:    e.source,
		Timestamp: time.Now().UTC(),
		Version:   DefaultVersion,
		Flags: EventFlags{
			Audit: true,
		},
	}

	for _, opt := range opts {
		opt(&evt)
	}

	return e.publisher.Publish(ctx, e.stream, evt)
}
