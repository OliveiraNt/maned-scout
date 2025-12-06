package repository

import (
	"os"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/registry"
)

type repoTestFactory struct{}

func (f repoTestFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
	return &dummyClient{}, nil
}

type dummyClient struct{}

func (d *dummyClient) IsHealthy() bool                           { return true }
func (d *dummyClient) ListTopics(_ bool) (map[string]int, error) { return map[string]int{}, nil }
func (d *dummyClient) GetClusterInfo() (*domain.Cluster, error)  { return nil, nil }
func (d *dummyClient) GetClusterStats() (*domain.ClusterStats, error) {
	return &domain.ClusterStats{}, nil
}
func (d *dummyClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (d *dummyClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (d *dummyClient) GetTopicDetail(_ string) (*domain.TopicDetail, error)       { return nil, nil }
func (d *dummyClient) CreateTopic(_ domain.CreateTopicRequest) error              { return nil }
func (d *dummyClient) DeleteTopic(_ string) error                                 { return nil }
func (d *dummyClient) UpdateTopicConfig(_ string, _ domain.UpdateTopicConfigRequest) error {
	return nil
}
func (d *dummyClient) IncreasePartitions(_ string, _ domain.IncreasePartitionsRequest) error {
	return nil
}
func (d *dummyClient) Close() {}

func TestSaveFindDelete(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

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
	path := tdir + "/config.yml"
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

func TestGetClient(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("get non-existent client", func(t *testing.T) {
		_, ok := repo.GetClient("nonexistent")
		if ok {
			t.Error("expected client not found")
		}
	})

	t.Run("get existing client", func(t *testing.T) {
		cfg := config.ClusterConfig{Name: "test", Brokers: []string{"localhost:9092"}}
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		client, ok := repo.GetClient("test")
		if !ok {
			t.Error("expected client to be found")
		}
		if client == nil {
			t.Error("expected non-nil client")
		}
	})
}

func TestFindByName(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("find non-existent cluster", func(t *testing.T) {
		_, ok := repo.FindByName("nonexistent")
		if ok {
			t.Error("expected cluster not found")
		}
	})

	t.Run("find existing cluster", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:     "test",
			Brokers:  []string{"localhost:9092"},
			ClientID: "test-client",
		}
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		found, ok := repo.FindByName("test")
		if !ok {
			t.Error("expected cluster to be found")
		}
		if found.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", found.Name)
		}
		if found.ClientID != "test-client" {
			t.Errorf("expected client_id 'test-client', got '%s'", found.ClientID)
		}
	})
}

func TestFindAll(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("find all with empty repository", func(t *testing.T) {
		all := repo.FindAll()
		if len(all) != 0 {
			t.Errorf("expected 0 clusters, got %d", len(all))
		}
	})

	t.Run("find all with multiple clusters", func(t *testing.T) {
		clusters := []config.ClusterConfig{
			{Name: "dev", Brokers: []string{"localhost:9092"}},
			{Name: "staging", Brokers: []string{"staging:9092"}},
			{Name: "prod", Brokers: []string{"prod:9092"}},
		}

		for _, c := range clusters {
			if err := repo.Save(c); err != nil {
				t.Fatalf("save %s: %v", c.Name, err)
			}
		}

		all := repo.FindAll()
		if len(all) != 3 {
			t.Errorf("expected 3 clusters, got %d", len(all))
		}

		// Verify it returns a copy, not the original
		all[0].Name = "modified"
		original := repo.FindAll()
		if original[0].Name == "modified" {
			t.Error("FindAll should return a copy, not the original")
		}
	})
}

func TestSaveUpdate(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("save new cluster", func(t *testing.T) {
		cfg := config.ClusterConfig{Name: "new", Brokers: []string{"localhost:9092"}}
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		found, ok := repo.FindByName("new")
		if !ok {
			t.Error("expected cluster to be found after save")
		}
		if found.Name != "new" {
			t.Errorf("expected name 'new', got '%s'", found.Name)
		}
	})

	t.Run("update existing cluster", func(t *testing.T) {
		// Save initial
		cfg := config.ClusterConfig{Name: "update", Brokers: []string{"old:9092"}}
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("initial save: %v", err)
		}

		// Update
		updated := config.ClusterConfig{Name: "update", Brokers: []string{"new:9092"}, ClientID: "new-client"}
		if err := repo.Save(updated); err != nil {
			t.Fatalf("update save: %v", err)
		}

		found, ok := repo.FindByName("update")
		if !ok {
			t.Error("expected cluster to be found after update")
		}
		if len(found.Brokers) != 1 || found.Brokers[0] != "new:9092" {
			t.Errorf("expected updated brokers, got %v", found.Brokers)
		}
		if found.ClientID != "new-client" {
			t.Errorf("expected client_id 'new-client', got '%s'", found.ClientID)
		}
	})
}

func TestDelete(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("delete non-existent cluster", func(t *testing.T) {
		err := repo.Delete("nonexistent")
		if err == nil {
			t.Error("expected error when deleting non-existent cluster")
		}
	})

	t.Run("delete existing cluster", func(t *testing.T) {
		cfg := config.ClusterConfig{Name: "todelete", Brokers: []string{"localhost:9092"}}
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		if err := repo.Delete("todelete"); err != nil {
			t.Fatalf("delete: %v", err)
		}

		if _, ok := repo.FindByName("todelete"); ok {
			t.Error("cluster should be deleted")
		}

		if _, ok := repo.GetClient("todelete"); ok {
			t.Error("client should be closed and removed")
		}
	})
}

func TestLoadFromFile(t *testing.T) {
	tdir := t.TempDir()
	path := tdir + "/config.yml"

	t.Run("load non-existent file", func(t *testing.T) {
		repo := NewClusterRepository(tdir+"/nonexistent.yml", repoTestFactory{})
		err := repo.LoadFromFile()
		if err == nil {
			t.Error("expected error when loading non-existent file")
		}
	})

	t.Run("load valid file", func(t *testing.T) {
		// Initialize logger for this test
		if registry.Logger == nil {
			registry.InitLogger()
		}

		// Create a config file manually
		cfg := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "cluster1", Brokers: []string{"broker1:9092"}},
				{Name: "cluster2", Brokers: []string{"broker2:9092"}},
			},
		}
		if err := config.WriteConfig(path, cfg); err != nil {
			t.Fatalf("write config: %v", err)
		}

		repo := NewClusterRepository(path, repoTestFactory{})
		if err := repo.LoadFromFile(); err != nil {
			t.Fatalf("load: %v", err)
		}

		all := repo.FindAll()
		if len(all) != 2 {
			t.Errorf("expected 2 clusters, got %d", len(all))
		}

		if _, ok := repo.GetClient("cluster1"); !ok {
			t.Error("expected client for cluster1")
		}
		if _, ok := repo.GetClient("cluster2"); !ok {
			t.Error("expected client for cluster2")
		}
	})
}

func TestReconcile(t *testing.T) {
	// Initialize logger for reconcile tests
	if registry.Logger == nil {
		registry.InitLogger()
	}

	tdir := t.TempDir()
	path := tdir + "/config.yml"
	repo := NewClusterRepository(path, repoTestFactory{})

	t.Run("reconcile creates new clients", func(t *testing.T) {
		cfg := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "new1", Brokers: []string{"broker1:9092"}},
				{Name: "new2", Brokers: []string{"broker2:9092"}},
			},
		}

		if err := repo.reconcile(cfg); err != nil {
			t.Fatalf("reconcile: %v", err)
		}

		if _, ok := repo.GetClient("new1"); !ok {
			t.Error("expected client for new1")
		}
		if _, ok := repo.GetClient("new2"); !ok {
			t.Error("expected client for new2")
		}
	})

	t.Run("reconcile removes old clients", func(t *testing.T) {
		// Add a cluster
		_ = repo.Save(config.ClusterConfig{Name: "old", Brokers: []string{"old:9092"}})

		// Reconcile with config that doesn't include "old"
		cfg := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "new", Brokers: []string{"new:9092"}},
			},
		}

		if err := repo.reconcile(cfg); err != nil {
			t.Fatalf("reconcile: %v", err)
		}

		if _, ok := repo.GetClient("old"); ok {
			t.Error("client for 'old' should be removed")
		}
		if _, ok := repo.GetClient("new"); !ok {
			t.Error("expected client for 'new'")
		}
	})
}

// Test equality helper functions
func TestEqualStrings(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"both nil", nil, nil, true},
		{"same order", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different order", []string{"a", "b"}, []string{"b", "a"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different values", []string{"a", "b"}, []string{"a", "c"}, false},
		{"duplicates handled", []string{"a", "a"}, []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalStrings(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalStrings(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestEqualTLS(t *testing.T) {
	tests := []struct {
		name     string
		a        *config.TLSConfig
		b        *config.TLSConfig
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", nil, &config.TLSConfig{}, false},
		{"identical", &config.TLSConfig{Enabled: true, CAFile: "ca.pem"}, &config.TLSConfig{Enabled: true, CAFile: "ca.pem"}, true},
		{"different enabled", &config.TLSConfig{Enabled: true}, &config.TLSConfig{Enabled: false}, false},
		{"different ca file", &config.TLSConfig{CAFile: "ca1.pem"}, &config.TLSConfig{CAFile: "ca2.pem"}, false},
		{"different cert file", &config.TLSConfig{CertFile: "cert1.pem"}, &config.TLSConfig{CertFile: "cert2.pem"}, false},
		{"different key file", &config.TLSConfig{KeyFile: "key1.pem"}, &config.TLSConfig{KeyFile: "key2.pem"}, false},
		{"different insecure", &config.TLSConfig{InsecureSkipVerify: true}, &config.TLSConfig{InsecureSkipVerify: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalTLS(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalTLS() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqualSASL(t *testing.T) {
	tests := []struct {
		name     string
		a        *config.SASLConfig
		b        *config.SASLConfig
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", nil, &config.SASLConfig{}, false},
		{"identical", &config.SASLConfig{Mechanism: "PLAIN", Username: "user", Password: "pass"}, &config.SASLConfig{Mechanism: "PLAIN", Username: "user", Password: "pass"}, true},
		{"different mechanism", &config.SASLConfig{Mechanism: "PLAIN"}, &config.SASLConfig{Mechanism: "SCRAM-SHA-256"}, false},
		{"different username", &config.SASLConfig{Username: "user1"}, &config.SASLConfig{Username: "user2"}, false},
		{"different password", &config.SASLConfig{Password: "pass1"}, &config.SASLConfig{Password: "pass2"}, false},
		{"different username env", &config.SASLConfig{UsernameEnv: "ENV1"}, &config.SASLConfig{UsernameEnv: "ENV2"}, false},
		{"different password env", &config.SASLConfig{PasswordEnv: "ENV1"}, &config.SASLConfig{PasswordEnv: "ENV2"}, false},
		{"different scram algorithm", &config.SASLConfig{ScramAlgorithm: "SHA-256"}, &config.SASLConfig{ScramAlgorithm: "SHA-512"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalSASL(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalSASL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqualAWS(t *testing.T) {
	tests := []struct {
		name     string
		a        *config.AWSConfig
		b        *config.AWSConfig
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", nil, &config.AWSConfig{}, false},
		{"identical", &config.AWSConfig{IAM: true, Region: "us-east-1"}, &config.AWSConfig{IAM: true, Region: "us-east-1"}, true},
		{"different iam", &config.AWSConfig{IAM: true}, &config.AWSConfig{IAM: false}, false},
		{"different region", &config.AWSConfig{Region: "us-east-1"}, &config.AWSConfig{Region: "us-west-2"}, false},
		{"different access key env", &config.AWSConfig{AccessKeyEnv: "KEY1"}, &config.AWSConfig{AccessKeyEnv: "KEY2"}, false},
		{"different secret key env", &config.AWSConfig{SecretKeyEnv: "SECRET1"}, &config.AWSConfig{SecretKeyEnv: "SECRET2"}, false},
		{"different session token env", &config.AWSConfig{SessionTokenEnv: "TOKEN1"}, &config.AWSConfig{SessionTokenEnv: "TOKEN2"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalAWS(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalAWS() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqualOptions(t *testing.T) {
	tests := []struct {
		name     string
		a        map[string]string
		b        map[string]string
		expected bool
	}{
		{"both empty", map[string]string{}, map[string]string{}, true},
		{"both nil", nil, nil, true},
		{"identical", map[string]string{"key1": "val1"}, map[string]string{"key1": "val1"}, true},
		{"different length", map[string]string{"key1": "val1"}, map[string]string{"key1": "val1", "key2": "val2"}, false},
		{"different values", map[string]string{"key1": "val1"}, map[string]string{"key1": "val2"}, false},
		{"different keys", map[string]string{"key1": "val1"}, map[string]string{"key2": "val1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalOptions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalOptions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClusterConfigEqual(t *testing.T) {
	t.Run("identical configs", func(t *testing.T) {
		a := config.ClusterConfig{
			Name:     "test",
			Brokers:  []string{"localhost:9092"},
			ClientID: "test-client",
		}
		b := config.ClusterConfig{
			Name:     "test",
			Brokers:  []string{"localhost:9092"},
			ClientID: "test-client",
		}

		if !clusterConfigEqual(a, b) {
			t.Error("expected identical configs to be equal")
		}
	})

	t.Run("different brokers", func(t *testing.T) {
		a := config.ClusterConfig{Brokers: []string{"localhost:9092"}}
		b := config.ClusterConfig{Brokers: []string{"other:9092"}}

		if clusterConfigEqual(a, b) {
			t.Error("expected different brokers to not be equal")
		}
	})

	t.Run("different client id", func(t *testing.T) {
		a := config.ClusterConfig{ClientID: "client1"}
		b := config.ClusterConfig{ClientID: "client2"}

		if clusterConfigEqual(a, b) {
			t.Error("expected different client ids to not be equal")
		}
	})

	t.Run("different tls", func(t *testing.T) {
		a := config.ClusterConfig{TLS: &config.TLSConfig{Enabled: true}}
		b := config.ClusterConfig{TLS: &config.TLSConfig{Enabled: false}}

		if clusterConfigEqual(a, b) {
			t.Error("expected different tls to not be equal")
		}
	})

	t.Run("different sasl", func(t *testing.T) {
		a := config.ClusterConfig{SASL: &config.SASLConfig{Mechanism: "PLAIN"}}
		b := config.ClusterConfig{SASL: &config.SASLConfig{Mechanism: "SCRAM-SHA-256"}}

		if clusterConfigEqual(a, b) {
			t.Error("expected different sasl to not be equal")
		}
	})

	t.Run("different aws", func(t *testing.T) {
		a := config.ClusterConfig{AWS: &config.AWSConfig{Region: "us-east-1"}}
		b := config.ClusterConfig{AWS: &config.AWSConfig{Region: "us-west-2"}}

		if clusterConfigEqual(a, b) {
			t.Error("expected different aws to not be equal")
		}
	})

	t.Run("different options", func(t *testing.T) {
		a := config.ClusterConfig{Options: map[string]string{"key": "val1"}}
		b := config.ClusterConfig{Options: map[string]string{"key": "val2"}}

		if clusterConfigEqual(a, b) {
			t.Error("expected different options to not be equal")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	// Test concurrent reads and writes
	t.Run("concurrent save and find", func(t *testing.T) {
		done := make(chan bool)

		// Writer goroutine
		go func() {
			for i := 0; i < 10; i++ {
				cfg := config.ClusterConfig{
					Name:    "concurrent",
					Brokers: []string{"localhost:9092"},
				}
				_ = repo.Save(cfg)
			}
			done <- true
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < 10; i++ {
				_ = repo.FindAll()
				_, _ = repo.FindByName("concurrent")
				_, _ = repo.GetClient("concurrent")
			}
			done <- true
		}()

		<-done
		<-done
	})
}

func TestNewClusterRepository(t *testing.T) {
	t.Run("create new repository", func(t *testing.T) {
		repo := NewClusterRepository("/tmp/config.yml", repoTestFactory{})
		if repo == nil {
			t.Error("expected non-nil repository")
		}
		if repo.clients == nil {
			t.Error("expected initialized clients map")
		}
		if repo.configPath != "/tmp/config.yml" {
			t.Errorf("expected config path '/tmp/config.yml', got '%s'", repo.configPath)
		}
	})
}

func TestSaveWithComplexConfig(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("save cluster with all options", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:     "complex",
			Brokers:  []string{"broker1:9092", "broker2:9092"},
			ClientID: "complex-client",
			TLS: &config.TLSConfig{
				Enabled:            true,
				CAFile:             "ca.pem",
				CertFile:           "cert.pem",
				KeyFile:            "key.pem",
				InsecureSkipVerify: false,
			},
			SASL: &config.SASLConfig{
				Mechanism: "SCRAM-SHA-256",
				Username:  "admin",
				Password:  "secret",
			},
			AWS: &config.AWSConfig{
				IAM:    true,
				Region: "us-east-1",
			},
			Options: map[string]string{
				"request.timeout.ms": "30000",
			},
		}

		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		found, ok := repo.FindByName("complex")
		if !ok {
			t.Fatal("expected cluster to be found")
		}

		// Verify all fields
		if found.Name != "complex" {
			t.Errorf("expected name 'complex', got '%s'", found.Name)
		}
		if len(found.Brokers) != 2 {
			t.Errorf("expected 2 brokers, got %d", len(found.Brokers))
		}
		if found.TLS == nil || !found.TLS.Enabled {
			t.Error("expected TLS to be enabled")
		}
		if found.SASL == nil || found.SASL.Mechanism != "SCRAM-SHA-256" {
			t.Error("expected SASL mechanism to be SCRAM-SHA-256")
		}
		if found.AWS == nil || !found.AWS.IAM {
			t.Error("expected AWS IAM to be enabled")
		}
		if found.Options["request.timeout.ms"] != "30000" {
			t.Error("expected options to be preserved")
		}
	})
}

func TestWatch(t *testing.T) {
	// Initialize logger
	if registry.Logger == nil {
		registry.InitLogger()
	}

	tdir := t.TempDir()
	path := tdir + "/config.yml"

	t.Run("watch valid directory", func(t *testing.T) {
		// Create initial config file
		cfg := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "initial", Brokers: []string{"localhost:9092"}},
			},
		}
		if err := config.WriteConfig(path, cfg); err != nil {
			t.Fatalf("write config: %v", err)
		}

		repo := NewClusterRepository(path, repoTestFactory{})
		if err := repo.LoadFromFile(); err != nil {
			t.Fatalf("load: %v", err)
		}

		if err := repo.Watch(); err != nil {
			t.Fatalf("watch: %v", err)
		}

		// Cleanup
		if repo.watcher != nil {
			repo.watcher.Close()
		}
	})

	t.Run("watch invalid path", func(t *testing.T) {
		repo := NewClusterRepository("/nonexistent/path/config.yml", repoTestFactory{})
		err := repo.Watch()
		if err == nil {
			t.Error("expected error when watching invalid path")
		}
	})
}

func TestReconcileWithConfigChange(t *testing.T) {
	// Initialize logger
	if registry.Logger == nil {
		registry.InitLogger()
	}

	tdir := t.TempDir()
	path := tdir + "/config.yml"

	t.Run("reconcile with config update", func(t *testing.T) {
		repo := NewClusterRepository(path, repoTestFactory{})

		// Create initial config with one cluster
		cfg1 := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "test", Brokers: []string{"localhost:9092"}},
			},
		}
		if err := repo.reconcile(cfg1); err != nil {
			t.Fatalf("initial reconcile: %v", err)
		}

		// Verify client exists
		_, ok := repo.GetClient("test")
		if !ok {
			t.Fatal("expected client to exist after initial reconcile")
		}

		// Update config with different brokers
		cfg2 := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "test", Brokers: []string{"localhost:9093", "localhost:9094"}},
				{Name: "new", Brokers: []string{"localhost:9095"}},
			},
		}

		// Reconcile with new config
		if err := repo.reconcile(cfg2); err != nil {
			t.Fatalf("second reconcile: %v", err)
		}

		// Verify both clients exist
		_, ok = repo.GetClient("test")
		if !ok {
			t.Fatal("expected test client to exist after reconcile")
		}

		_, ok = repo.GetClient("new")
		if !ok {
			t.Fatal("expected new client to exist after reconcile")
		}
	})
}

func TestWriteToFile(t *testing.T) {
	tdir := t.TempDir()
	path := tdir + "/subdir/config.yml"

	t.Run("write to file creates directories", func(t *testing.T) {
		repo := NewClusterRepository(path, repoTestFactory{})
		cfg := config.ClusterConfig{
			Name:    "test",
			Brokers: []string{"localhost:9092"},
		}

		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("expected file to be created")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	tdir := t.TempDir()
	repo := NewClusterRepository(tdir+"/config.yml", repoTestFactory{})

	t.Run("save cluster with empty name", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "",
			Brokers: []string{"localhost:9092"},
		}

		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		_, ok := repo.FindByName("")
		if !ok {
			t.Error("should be able to save cluster with empty name")
		}
	})

	t.Run("save cluster with nil slices and maps", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "nil-test",
			Brokers: nil,
			Options: nil,
		}

		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		found, ok := repo.FindByName("nil-test")
		if !ok {
			t.Fatal("expected cluster to be found")
		}
		if found.Name != "nil-test" {
			t.Errorf("expected name 'nil-test', got '%s'", found.Name)
		}
	})

	t.Run("delete and re-add cluster", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "readd",
			Brokers: []string{"localhost:9092"},
		}

		// Add
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("save: %v", err)
		}

		// Delete
		if err := repo.Delete("readd"); err != nil {
			t.Fatalf("delete: %v", err)
		}

		// Re-add
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("re-save: %v", err)
		}

		found, ok := repo.FindByName("readd")
		if !ok {
			t.Error("expected re-added cluster to be found")
		}
		if found.Name != "readd" {
			t.Errorf("expected name 'readd', got '%s'", found.Name)
		}
	})

	t.Run("multiple saves same cluster updates client", func(t *testing.T) {
		cfg := config.ClusterConfig{
			Name:    "multi",
			Brokers: []string{"localhost:9092"},
		}

		// First save
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("first save: %v", err)
		}

		// Second save with same config
		if err := repo.Save(cfg); err != nil {
			t.Fatalf("second save: %v", err)
		}

		all := repo.FindAll()
		count := 0
		for _, c := range all {
			if c.Name == "multi" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected 1 cluster named 'multi', got %d", count)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	// Initialize logger
	if registry.Logger == nil {
		registry.InitLogger()
	}

	t.Run("factory error on save", func(t *testing.T) {
		// Use a factory that returns errors
		errorFactory := &errorTestFactory{shouldError: true}
		tdir := t.TempDir()
		repo := NewClusterRepository(tdir+"/config.yml", errorFactory)

		cfg := config.ClusterConfig{
			Name:    "error-test",
			Brokers: []string{"localhost:9092"},
		}

		err := repo.Save(cfg)
		if err == nil {
			t.Error("expected error from factory")
		}
	})

	t.Run("factory error on reconcile", func(t *testing.T) {
		errorFactory := &errorTestFactory{shouldError: true}
		tdir := t.TempDir()
		repo := NewClusterRepository(tdir+"/config.yml", errorFactory)

		cfg := config.FileConfig{
			Clusters: []config.ClusterConfig{
				{Name: "test", Brokers: []string{"localhost:9092"}},
			},
		}

		// Should not error out, just log the error
		if err := repo.reconcile(cfg); err != nil {
			t.Fatalf("reconcile should not error: %v", err)
		}

		// Client should not be created
		if _, ok := repo.GetClient("test"); ok {
			t.Error("client should not exist due to factory error")
		}
	})
}

// Error factory for testing error cases
type errorTestFactory struct {
	shouldError bool
}

func (f *errorTestFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
	if f.shouldError {
		return nil, &testError{msg: "factory error"}
	}
	return &dummyClient{}, nil
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
