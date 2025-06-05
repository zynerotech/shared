package kafka

import (
	"context"
	json "github.com/bytedance/sonic"
	"github.com/zynerotech/shared/transport"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

// KafkaEventPublisher реализует интерфейс Publisher для отправки событий в Kafka.
type KafkaEventPublisher struct {
	producer transport.Producer // Используем интерфейс Producer из pkg/transport
	topic    string
}

// NewKafkaEventPublisher создает новый экземпляр KafkaEventPublisher.
func NewKafkaEventPublisher(p transport.Producer, topic string) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		producer: p,
		topic:    topic,
	}
}

// Publish сериализует полезную нагрузку и отправляет ее в Kafka, обернув в Envelope.
func (kep *KafkaEventPublisher) Publish(ctx context.Context, eventType string, eventID string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling payload")
		return err // Ошибка маршалинга полезной нагрузки
	}

	// Если eventID не предоставлен, генерируем новый UUID.
	if eventID == "" {
		eventID = uuid.NewString()
	}

	envelope := transport.Envelope{
		EventID:    eventID,
		EventType:  eventType,
		OccurredAt: time.Now().UTC(), // Важно использовать UTC для консистентности
		Payload:    payloadBytes,     // json.RawMessage, поэтому присваиваем напрямую
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling event envelope") // Ошибка маршалинга конверта
		return err
	}

	// В качестве ключа Kafka используем EventID для обеспечения возможного упорядочивания
	// или партиционирования по ID события, если это необходимо.
	return kep.producer.Publish(ctx, kep.topic, envelope.EventID, envelopeBytes)
}
