package application

import (
	"context"
	"errors"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
)

type mockRepo struct {
	saveErr   error
	deleteErr error
	items     map[string]config.ClusterConfig
}

func newMockRepo() *mockRepo {
	return &mockRepo{items: make(map[string]config.ClusterConfig)}
}

func (m *mockRepo) Save(cfg config.ClusterConfig) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.items[cfg.Name] = cfg
	return nil
}
func (m *mockRepo) Delete(name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.items, name)
	return nil
}
func (m *mockRepo) FindByName(name string) (config.ClusterConfig, bool) {
	cfg, ok := m.items[name]
	return cfg, ok
}
func (m *mockRepo) FindAll() []config.ClusterConfig {
	out := make([]config.ClusterConfig, 0, len(m.items))
	for _, v := range m.items {
		out = append(out, v)
	}
	return out
}
func (m *mockRepo) Watch() error { return nil }

type mockClientFactory struct {
	ret domain.KafkaClient
	err error
}

func (f mockClientFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
	return f.ret, f.err
}

type mockClient struct {
	healthy  bool
	stats    *domain.ClusterStats
	statsErr error
}

func (c *mockClient) IsHealthy() bool                                            { return c.healthy }
func (c *mockClient) ListTopics(_ bool) (map[string]int, error)                  { return nil, nil }
func (c *mockClient) GetClusterInfo() (*domain.Cluster, error)                   { return nil, nil }
func (c *mockClient) GetClusterStats() (*domain.ClusterStats, error)             { return c.stats, c.statsErr }
func (c *mockClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (c *mockClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (c *mockClient) GetTopicDetail(_ string) (*domain.TopicDetail, error)       { return nil, nil }
func (c *mockClient) CreateTopic(_ domain.CreateTopicRequest) error              { return nil }
func (c *mockClient) DeleteTopic(_ string) error                                 { return nil }
func (c *mockClient) UpdateTopicConfig(_ string, _ domain.UpdateTopicConfigRequest) error {
	return nil
}
func (c *mockClient) IncreasePartitions(_ string, _ domain.IncreasePartitionsRequest) error {
	return nil
}
func (c *mockClient) StreamMessages(_ context.Context, _ string, _ chan<- []byte) {}
func (c *mockClient) Close()                                                      {}

func TestListAndGetClusters(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	repo.items["prod"] = config.ClusterConfig{Name: "prod", Brokers: []string{"kafka:9092"}}

	svc := NewClusterService(repo, mockClientFactory{})
	all := svc.ListClusters()
	if len(all) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(all))
	}

	cfg, ok := svc.GetCluster("dev")
	if !ok {
		t.Fatalf("expected to find dev cluster")
	}
	if cfg.Name != "dev" {
		t.Fatalf("expected dev, got %s", cfg.Name)
	}
}

func TestAddClusterValidation(t *testing.T) {
	repo := newMockRepo()
	svc := NewClusterService(repo, mockClientFactory{})

	if err := svc.AddCluster(config.ClusterConfig{}); !errors.Is(err, ErrInvalidClusterConfig) {
		t.Fatalf("expected ErrInvalidClusterConfig, got %v", err)
	}
	if err := svc.AddCluster(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.FindByName("dev"); !ok {
		t.Fatalf("cluster should be saved")
	}
}

func TestUpdateClusterKeepsName(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	svc := NewClusterService(repo, mockClientFactory{})

	// Update without name should preserve provided key
	upd := config.ClusterConfig{Brokers: []string{"localhost:9093"}}
	if err := svc.UpdateCluster("dev", upd); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	cfg, ok := repo.FindByName("dev")
	if !ok {
		t.Fatalf("expected dev to exist")
	}
	if cfg.Name != "dev" {
		t.Fatalf("expected name 'dev', got %s", cfg.Name)
	}
	if cfg.Brokers[0] != "localhost:9093" {
		t.Fatalf("brokers not updated")
	}
}

func TestDeleteCluster(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	svc := NewClusterService(repo, mockClientFactory{})
	if err := svc.DeleteCluster("dev"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if _, ok := repo.FindByName("dev"); ok {
		t.Fatalf("cluster should be deleted")
	}
}

func TestGetClusterWithStats_NoClient(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	svc := NewClusterService(repo, mockClientFactory{})

	cluster, stats, err := svc.GetClusterWithStats("dev", nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cluster == nil || cluster.Name != "dev" {
		t.Fatalf("cluster mismatch")
	}
	if cluster.IsOnline {
		t.Fatalf("expected offline without client")
	}
	if stats != nil {
		t.Fatalf("expected nil stats")
	}
}

func TestGetClusterWithStats_HealthyWithStats(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	client := &mockClient{healthy: true, stats: &domain.ClusterStats{TotalTopics: 3}}
	svc := NewClusterService(repo, mockClientFactory{})

	cluster, stats, err := svc.GetClusterWithStats("dev", client)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !cluster.IsOnline {
		t.Fatalf("expected online")
	}
	if stats == nil || stats.TotalTopics != 3 {
		t.Fatalf("expected stats TotalTopics=3")
	}
}

func TestGetClusterWithStats_Unhealthy(t *testing.T) {
	repo := newMockRepo()
	repo.items["dev"] = config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	client := &mockClient{healthy: false}
	svc := NewClusterService(repo, mockClientFactory{})

	cluster, stats, err := svc.GetClusterWithStats("dev", client)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cluster.IsOnline {
		t.Fatalf("expected offline")
	}
	if stats != nil {
		t.Fatalf("expected nil stats when offline")
	}
}

func TestGetClusterWithStats_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewClusterService(repo, mockClientFactory{})
	_, _, err := svc.GetClusterWithStats("missing", nil)
	if !errors.Is(err, ErrClusterNotFound) {
		t.Fatalf("expected ErrClusterNotFound, got %v", err)
	}
}
