package kafka

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

type kafkaContainer struct {
	*kafka.KafkaContainer
	Brokers []string
}

func setupKafka(ctx context.Context) (*kafkaContainer, error) {
	container, err := kafka.Run(ctx,
		"confluentinc/cp-kafka:7.4.0",
		kafka.WithClusterID("test-cluster-id"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	brokers, err := container.Brokers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get brokers: %w", err)
	}

	return &kafkaContainer{
		KafkaContainer: container,
		Brokers:        brokers,
	}, nil
}

func getTestBrokers(t *testing.T) []string {
	ctx := context.Background()
	container, err := setupKafka(ctx)
	if err != nil {
		t.Fatalf("failed to setup kafka: %v", err)
	}

	t.Cleanup(func() {
		// Use a separate context for cleanup to ensure it runs even if the original context is cancelled
		cleanupCtx := context.Background()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	return container.Brokers
}
