package grpc

import (
	"testing"

	platformlogger "gitlab.com/zynero/shared/logger"
)

func TestNewServerNilOption(t *testing.T) {
	cfg := Config{Address: ":0"}
	l, err := platformlogger.New(platformlogger.Config{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if _, err := NewServer(cfg, l, nil); err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
}
