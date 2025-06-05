package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config представляет конфигурацию подключения к базе данных
type Config struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	DBName            string        `mapstructure:"dbname"`
	SSLMode           string        `mapstructure:"sslmode"`
	MaxConns          int           `mapstructure:"max_conns"`
	MinConns          int           `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

// Database представляет менеджер подключения к базе данных
type Database struct {
	config Config
	pool   *pgxpool.Pool
}

// New создает новый экземпляр менеджера подключения к базе данных
func New(cfg Config) (*Database, error) {
	// Формируем строку подключения
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Создаем конфигурацию пула соединений
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Настраиваем пул соединений
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = cfg.Timeout

	// Создаем пул соединений
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		config: cfg,
		pool:   pool,
	}, nil
}

// Close закрывает пул соединений
func (d *Database) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

// Pool возвращает пул соединений
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// Begin начинает транзакцию
func (d *Database) Begin(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctx)
}

// Exec выполняет запрос без возврата результатов
func (d *Database) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := d.pool.Exec(ctx, sql, args...)
	return err
}

// Query выполняет запрос с возвратом результатов
func (d *Database) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return d.pool.Query(ctx, sql, args...)
}

// QueryRow выполняет запрос с возвратом одной строки
func (d *Database) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, sql, args...)
}

// Ping проверяет подключение к базе данных
func (d *Database) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}
