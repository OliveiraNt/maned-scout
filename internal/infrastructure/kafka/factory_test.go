package kafka

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
)

func TestFactory_CreateClient(t *testing.T) {
	f := NewFactory()
	c, err := f.CreateClient(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c == nil {
		t.Fatalf("client should not be nil")
	}
}
