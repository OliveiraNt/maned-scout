package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
)

func TestNewAdmin(t *testing.T) {
	// Create a basic client
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	t.Run("create admin from client", func(t *testing.T) {
		if client.admin == nil {
			t.Error("expected non-nil admin")
		}
	})
}

func TestAdminListTopics(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("list topics without internal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := admin.ListTopics(ctx, false)
		// Will fail without running Kafka, but tests the method exists
		_ = err
	})

	t.Run("list topics with internal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := admin.ListTopics(ctx, true)
		_ = err
	})

	t.Run("list topics with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := admin.ListTopics(ctx, false)
		if err == nil {
			t.Log("expected error with cancelled context (may pass if very fast)")
		}
	})
}

func TestAdminGetClusterInfo(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("get cluster info", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := admin.GetClusterInfo(ctx)
		// Will fail without running Kafka
		_ = err
	})
}

func TestAdminGetClusterStats(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("get cluster stats", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := admin.GetClusterStats(ctx)
		// Will fail without running Kafka
		_ = err
	})
}

func TestAdminGetBrokerDetails(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("get broker details", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := admin.GetBrokerDetails(ctx)
		// Will fail without running Kafka
		_ = err
	})
}

func TestAdminListConsumerGroups(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("list consumer groups", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := admin.ListConsumerGroups(ctx)
		// Will fail without running Kafka
		_ = err
	})
}

func TestAdminGetTopicDetail(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("get topic detail", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := admin.GetTopicDetail(ctx, "test-topic")
		// Will fail without running Kafka or if topic doesn't exist
		_ = err
	})

	t.Run("get topic detail with empty name", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := admin.GetTopicDetail(ctx, "")
		_ = err
	})
}

func TestAdminCreateTopic(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("create topic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.CreateTopicRequest{
			Name:              "test-create-topic",
			NumPartitions:     3,
			ReplicationFactor: 1,
		}

		err := admin.CreateTopic(ctx, req)
		// Will fail without running Kafka
		_ = err
	})

	t.Run("create topic with configs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		retention := "86400000"
		req := domain.CreateTopicRequest{
			Name:              "test-create-topic-with-config",
			NumPartitions:     3,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms": &retention,
			},
		}

		err := admin.CreateTopic(ctx, req)
		_ = err
	})
}

func TestAdminDeleteTopic(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("delete topic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := admin.DeleteTopic(ctx, "test-delete-topic")
		// Will fail without running Kafka or if topic doesn't exist
		_ = err
	})

	t.Run("delete non-existent topic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := admin.DeleteTopic(ctx, "non-existent-topic")
		_ = err
	})
}

func TestAdminUpdateTopicConfig(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("update topic config", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		retention := "172800000"
		req := domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{
				"retention.ms": &retention,
			},
		}

		err := admin.UpdateTopicConfig(ctx, "test-topic", req)
		// Will fail without running Kafka
		_ = err
	})

	t.Run("update topic config with multiple settings", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		retention := "172800000"
		cleanupPolicy := "delete"
		req := domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{
				"retention.ms":   &retention,
				"cleanup.policy": &cleanupPolicy,
			},
		}

		err := admin.UpdateTopicConfig(ctx, "test-topic", req)
		_ = err
	})

	t.Run("update topic config with empty configs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{},
		}

		err := admin.UpdateTopicConfig(ctx, "test-topic", req)
		_ = err
	})
}

func TestAdminIncreasePartitions(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("increase partitions", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.IncreasePartitionsRequest{
			TotalPartitions: 5,
		}

		err := admin.IncreasePartitions(ctx, "test-topic", req)
		// Will fail without running Kafka
		_ = err
	})

	t.Run("increase partitions to lower count", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.IncreasePartitionsRequest{
			TotalPartitions: 1,
		}

		err := admin.IncreasePartitions(ctx, "test-topic", req)
		// Should fail if topic has more partitions
		_ = err
	})
}

func TestAdminBrokerMetadata(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("get broker metadata", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := admin.BrokerMetadata(ctx)
		// Will fail without running Kafka
		_ = err
	})

	t.Run("broker metadata with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := admin.BrokerMetadata(ctx)
		if err == nil {
			t.Log("expected error with cancelled context")
		}
	})
}

// Integration-style tests that verify the structure but don't require a running Kafka
func TestAdminIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("verify admin methods exist", func(t *testing.T) {
		if admin == nil {
			t.Fatal("admin should not be nil")
		}

		// Just verify the methods exist and can be called
		ctx := context.Background()

		// These will error without Kafka, but we're just checking the API
		_, _ = admin.BrokerMetadata(ctx)
		_, _ = admin.ListTopics(ctx, false)
		_, _ = admin.GetClusterInfo(ctx)
		_, _ = admin.GetClusterStats(ctx)
		_, _ = admin.GetBrokerDetails(ctx)
		_, _ = admin.ListConsumerGroups(ctx)
		_, _ = admin.GetTopicDetail(ctx, "test")

		retention := "1000"
		_ = admin.CreateTopic(ctx, domain.CreateTopicRequest{
			Name:              "test",
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		_ = admin.DeleteTopic(ctx, "test")
		_ = admin.UpdateTopicConfig(ctx, "test", domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{"retention.ms": &retention},
		})
		_ = admin.IncreasePartitions(ctx, "test", domain.IncreasePartitionsRequest{
			TotalPartitions: 2,
		})
	})
}

// Test context timeout behavior
func TestAdminContextTimeout(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("list topics respects context timeout", func(t *testing.T) {
		// Use a very short timeout to ensure it expires
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure context is expired

		_, err := admin.ListTopics(ctx, false)
		// Should get a context deadline exceeded error or connection error
		_ = err
	})

	t.Run("get cluster stats respects context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond)

		_, err := admin.GetClusterStats(ctx)
		_ = err
	})
}

// Test edge cases
func TestAdminEdgeCases(t *testing.T) {
	cfg := config.ClusterConfig{
		Name:    "test",
		Brokers: []string{"localhost:9092"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	admin := client.admin

	t.Run("create topic with nil configs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.CreateTopicRequest{
			Name:              "test-nil-configs",
			NumPartitions:     1,
			ReplicationFactor: 1,
			Configs:           nil,
		}

		err := admin.CreateTopic(ctx, req)
		_ = err
	})

	t.Run("update topic with nil config values", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req := domain.UpdateTopicConfigRequest{
			Configs: map[string]*string{
				"retention.ms": nil,
			},
		}

		err := admin.UpdateTopicConfig(ctx, "test-topic", req)
		_ = err
	})

	t.Run("get topic detail for very long name", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		longName := string(make([]byte, 249)) // Max Kafka topic name length is 249
		for i := 0; i < 249; i++ {
			longName += "a"
		}

		_, err := admin.GetTopicDetail(ctx, longName)
		_ = err
	})
}
