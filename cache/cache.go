package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

// Config представляет конфигурацию для кеша
type Config struct {
	Enabled  bool          `mapstructure:"enabled"`
	Host     string        `mapstructure:"host"`
	Password string        `mapstructure:"password"`
	Port     int           `mapstructure:"port"`
	DB       int           `mapstructure:"db"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// Cache определяет интерфейс для работы с кешем
type Cache interface {
	// Get получает значение по ключу
	Get(ctx context.Context, key string) ([]byte, error)
	// Set сохраняет значение по ключу с указанным TTL
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Delete удаляет значение по ключу
	Delete(ctx context.Context, key string) error
	// Marshal сериализует значение в байты
	Marshal(v any) ([]byte, error)
	// Unmarshal десериализует байты в значение
	Unmarshal(data []byte, v any) error
}

// New создает новый экземпляр кеша на основе конфигурации
func New(config Config) (Cache, error) {
	if !config.Enabled {
		return newNoopCache(), nil
	}
	return newRedisCache(config)
}

// redisCache реализует Cache с использованием Redis
type redisCache struct {
	client *redis.Client
	cfg    Config
}

func newRedisCache(config Config) (*redisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &redisCache{
		client: rdb,
		cfg:    config,
	}, nil
}

func (rc *redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := rc.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s from redis: %w", key, err)
	}
	return val, nil
}

func (rc *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := rc.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	actualTTL := rc.cfg.TTL
	if ttl > 0 {
		actualTTL = ttl
	}

	if err := rc.client.Set(ctx, key, data, actualTTL).Err(); err != nil {
		return fmt.Errorf("failed to set key %s in redis: %w", key, err)
	}
	return nil
}

func (rc *redisCache) Delete(ctx context.Context, key string) error {
	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s from redis: %w", key, err)
	}
	return nil
}

func (rc *redisCache) Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func (rc *redisCache) Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}

// noopCache реализует Cache с пустой реализацией
type noopCache struct{}

func newNoopCache() *noopCache {
	return &noopCache{}
}

func (nc *noopCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (nc *noopCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error {
	return nil
}

func (nc *noopCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (nc *noopCache) Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func (nc *noopCache) Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}
