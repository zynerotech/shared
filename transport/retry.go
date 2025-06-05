package transport

import (
	"errors"
	"time"
)

// RetryPolicy определяет политику повторных попыток
type RetryPolicy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// DefaultRetryPolicy возвращает политику retry по умолчанию
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		BaseDelay:     time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryableError определяет интерфейс для ошибок с информацией о возможности retry
type RetryableError interface {
	error
	IsRetryable() bool
	RetryAfter() time.Duration
}

// NonRetryableError создает ошибку, которая не должна повторяться
func NewNonRetryableError(err error) error {
	return &nonRetryableError{err: err}
}

type nonRetryableError struct {
	err error
}

func (e *nonRetryableError) Error() string {
	return e.err.Error()
}

func (e *nonRetryableError) Unwrap() error {
	return e.err
}

func (e *nonRetryableError) IsRetryable() bool {
	return false
}

func (e *nonRetryableError) RetryAfter() time.Duration {
	return 0
}

// TemporaryError создает временную ошибку, которая может быть повторена
func NewTemporaryError(err error, retryAfter time.Duration) error {
	return &temporaryError{
		err:        err,
		retryAfter: retryAfter,
	}
}

type temporaryError struct {
	err        error
	retryAfter time.Duration
}

func (e *temporaryError) Error() string {
	return e.err.Error()
}

func (e *temporaryError) Unwrap() error {
	return e.err
}

func (e *temporaryError) IsRetryable() bool {
	return true
}

func (e *temporaryError) RetryAfter() time.Duration {
	return e.retryAfter
}

// IsRetryableError проверяет, является ли ошибка повторяемой
func IsRetryableError(err error) bool {
	var retryableErr RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}
	// По умолчанию считаем ошибки повторяемыми
	return true
}
