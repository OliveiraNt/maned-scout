package kafka

import (
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
)

// Factory creates Kafka clients from configuration.
type Factory struct{}

// NewFactory creates a new client factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateClient creates a new Kafka client from configuration.
func (f *Factory) CreateClient(cfg config.ClusterConfig) (domain.KafkaClient, error) {
	return NewClient(cfg)
}
