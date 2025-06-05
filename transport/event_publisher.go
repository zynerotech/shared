package transport

import (
	"context"
)

// EventPublisher определяет интерфейс для публикации событий.
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, eventID string, payload any) error
}
