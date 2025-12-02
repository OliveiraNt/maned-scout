package repository

import (
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
)

type repoTestFactory struct{}

func (f repoTestFactory) CreateClient(cfg config.ClusterConfig) (domain.KafkaClient, error) {
	return &dummyClient{}, nil
}

type dummyClient struct{}

func (d *dummyClient) IsHealthy() bool                          { return true }
func (d *dummyClient) ListTopics(bool) (map[string]int, error)  { return map[string]int{}, nil }
func (d *dummyClient) GetClusterInfo() (*domain.Cluster, error) { return nil, nil }
func (d *dummyClient) GetClusterStats() (*domain.ClusterStats, error) {
	return &domain.ClusterStats{}, nil
}
func (d *dummyClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (d *dummyClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (d *dummyClient) Close()                                                     {}

func TestSaveFindDelete(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/clusters.yml", repoTestFactory{})

	// Save two clusters
	c1 := config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}}
	c2 := config.ClusterConfig{Name: "prod", Brokers: []string{"kafka:9092"}}
	if err := repo.Save(c1); err != nil {
		t.Fatalf("save c1: %v", err)
	}
	if err := repo.Save(c2); err != nil {
		t.Fatalf("save c2: %v", err)
	}

	// FindByName
	cfg, ok := repo.FindByName("dev")
	if !ok || cfg.Name != "dev" {
		t.Fatalf("FindByName dev failed: %+v", cfg)
	}

	// FindAll
	all := repo.FindAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(all))
	}

	// Delete
	if err := repo.Delete("dev"); err != nil {
		t.Fatalf("delete dev: %v", err)
	}
	if _, ok := repo.FindByName("dev"); ok {
		t.Fatalf("dev should be removed")
	}
	all = repo.FindAll()
	if len(all) != 1 || all[0].Name != "prod" {
		t.Fatalf("unexpected remaining: %+v", all)
	}
}

func TestLoadFromFileReconcile(t *testing.T) {
	tdir := t.TempDir()
	path := tdir + "/clusters.yml"
	repo := NewClusterRepository(path, repoTestFactory{})

	// write initial config via Save
	_ = repo.Save(config.ClusterConfig{Name: "a", Brokers: []string{"b1:9092"}})
	_ = repo.Save(config.ClusterConfig{Name: "b", Brokers: []string{"b2:9092"}})

	// reload from file
	if err := repo.LoadFromFile(); err != nil {
		t.Fatalf("load: %v", err)
	}

	// Ensure clients exist for both
	if _, ok := repo.GetClient("a"); !ok {
		t.Fatalf("client for a missing")
	}
	if _, ok := repo.GetClient("b"); !ok {
		t.Fatalf("client for b missing")
	}

	// Delete one cluster from file by calling Delete
	if err := repo.Delete("a"); err != nil {
		t.Fatalf("delete a: %v", err)
	}

	// Re-load and ensure only b remains
	if err := repo.LoadFromFile(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := repo.GetClient("a"); ok {
		t.Fatalf("client for a should be gone")
	}
	if _, ok := repo.GetClient("b"); !ok {
		t.Fatalf("client for b should remain")
	}
}
