package transport

import (
	"context"
	"io"
)

// Producer определяет интерфейс для публикации сообщений в транспорт
type Producer interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	io.Closer // Добавляем интерфейс для graceful shutdown
}
