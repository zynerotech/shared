package transport

import (
	"context"
	"io"
	"time"
)

// Consumer определяет интерфейс для потребления сообщений из транспорта
type Consumer interface {
	// Run запускает consumer и блокирует выполнение до получения сигнала остановки
	Run(ctx context.Context) error

	// Stop инициирует graceful shutdown
	Stop()

	// Wait ожидает завершения работы consumer с таймаутом
	Wait(timeout time.Duration) error

	// Closer для освобождения ресурсов
	io.Closer
}
