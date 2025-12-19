package repository

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/infrastructure/kafka"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/fsnotify/fsnotify"
)

// ClusterRepository manages cluster configurations and their clients.
type ClusterRepository struct {
	mu         sync.RWMutex
	clients    map[string]domain.KafkaClient
	configData config.FileConfig
	configPath string
	watcher    *fsnotify.Watcher
	factory    domain.ClientFactory
}

// NewClusterRepository creates a new cluster repository.
func NewClusterRepository(configPath string, factory domain.ClientFactory) *ClusterRepository {
	return &ClusterRepository{
		clients:    make(map[string]domain.KafkaClient),
		configPath: configPath,
		factory:    factory,
	}
}

// LoadFromFile loads configuration from file.
func (r *ClusterRepository) LoadFromFile() error {
	cfg, err := config.ReadConfig(r.configPath)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.configData = cfg
	r.mu.Unlock()

	return r.reconcile(cfg)
}

// Save persists a cluster configuration.
func (r *ClusterRepository) Save(cfg config.ClusterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, err := r.factory.CreateClient(cfg)
	if err != nil {
		return err
	}
	if old, ok := r.clients[cfg.Name]; ok {
		old.Close()
	}
	r.clients[cfg.Name] = client
	found := false
	for i := range r.configData.Clusters {
		if r.configData.Clusters[i].Name == cfg.Name {
			r.configData.Clusters[i] = cfg
			found = true
			break
		}
	}
	if !found {
		r.configData.Clusters = append(r.configData.Clusters, cfg)
	}
	return r.writeToFile()
}

// Delete removes a cluster configuration by name.
func (r *ClusterRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, ok := r.clients[name]
	if !ok {
		return errors.New("cluster not found")
	}

	client.Close()
	delete(r.clients, name)
	idx := -1
	for i := range r.configData.Clusters {
		if r.configData.Clusters[i].Name == name {
			idx = i
			break
		}
	}
	if idx >= 0 {
		r.configData.Clusters = append(r.configData.Clusters[:idx], r.configData.Clusters[idx+1:]...)
	}
	return r.writeToFile()
}

// FindByName retrieves a cluster configuration by name
func (r *ClusterRepository) FindByName(name string) (config.ClusterConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.configData.Clusters {
		if c.Name == name {
			return c, true
		}
	}
	return config.ClusterConfig{}, false
}

// FindAll retrieves all cluster configurations
func (r *ClusterRepository) FindAll() []config.ClusterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]config.ClusterConfig, len(r.configData.Clusters))
	copy(out, r.configData.Clusters)
	return out
}

// GetClient returns a Kafka client for the given cluster name
func (r *ClusterRepository) GetClient(name string) (domain.KafkaClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, ok := r.clients[name]
	return client, ok
}

// Watch sets a fsnotify watcher on the file for hot reload
func (r *ClusterRepository) Watch() error {
	abs, err := filepath.Abs(r.configPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(abs)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := w.Add(dir); err != nil {
		return err
	}

	r.watcher = w

	go func() {
		for {
			select {
			case event, ok := <-r.watcher.Events:
				if !ok {
					return
				}
				abs := r.configPath
				utils.Logger.Info("config file changed", "path", abs, "event", event)
				if err := r.LoadFromFile(); err != nil {
					utils.Logger.Error("failed to reload config", "err", err)
				}
			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}
				utils.Logger.Error("fsnotify error", "err", err)
			}
		}
	}()
	return nil
}

// reconcile synchronizes clients with configuration
func (r *ClusterRepository) reconcile(cfg config.FileConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing := make(map[string]struct{})
	for _, c := range cfg.Clusters {
		existing[c.Name] = struct{}{}

		cur, ok := r.clients[c.Name]
		if !ok {
			client, err := r.factory.CreateClient(c)
			if err != nil {
				utils.Logger.Error("failed to create client", "cluster", c.Name, "err", err)
				continue
			}
			r.clients[c.Name] = client
			utils.Logger.Info("client created", "cluster", c.Name)
			continue
		}

		if kafkaClient, ok := cur.(*kafka.Client); ok {
			if !clusterConfigEqual(kafkaClient.GetConfig(), c) {
				cur.Close()
				client, err := r.factory.CreateClient(c)
				if err != nil {
					utils.Logger.Error("failed to recreate client", "cluster", c.Name, "err", err)
					continue
				}
				r.clients[c.Name] = client
				utils.Logger.Info("client recreated", "cluster", c.Name)
			}
		}
	}

	for name, client := range r.clients {
		if _, ok := existing[name]; !ok {
			client.Close()
			delete(r.clients, name)
		}
	}

	return nil
}

// writeToFile persists current in-memory config to file
func (r *ClusterRepository) writeToFile() error {
	dir := filepath.Dir(r.configPath)
	_ = os.MkdirAll(dir, 0755)
	return config.WriteConfig(r.configPath, r.configData)
}

// Close releases all resources held by the ClusterRepository, including watcher and Kafka clients.
func (r *ClusterRepository) Close() {
	utils.Logger.Info("Closing repository")
	r.watcher.Close()
	for k, client := range r.clients {
		utils.Logger.Info("Closing client", "cluster", k)
		client.Close()
	}
}

// clusterConfigEqual compares cluster configurations
func clusterConfigEqual(a, b config.ClusterConfig) bool {
	if !equalStrings(a.Brokers, b.Brokers) || a.ClientID != b.ClientID {
		return false
	}
	if !equalTLS(a.TLS, b.TLS) || !equalSASL(a.SASL, b.SASL) {
		return false
	}
	if !equalAWS(a.AWS, b.AWS) || !equalOptions(a.Options, b.Options) {
		return false
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int)
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		if m[s] == 0 {
			return false
		}
		m[s]--
	}
	return true
}

func equalTLS(a, b *config.TLSConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Enabled == b.Enabled && a.CAFile == b.CAFile &&
		a.CertFile == b.CertFile && a.KeyFile == b.KeyFile &&
		a.InsecureSkipVerify == b.InsecureSkipVerify
}

func equalSASL(a, b *config.SASLConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Mechanism == b.Mechanism && a.Username == b.Username &&
		a.Password == b.Password && a.UsernameEnv == b.UsernameEnv &&
		a.PasswordEnv == b.PasswordEnv && a.ScramAlgorithm == b.ScramAlgorithm
}

func equalAWS(a, b *config.AWSConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.IAM == b.IAM && a.Region == b.Region &&
		a.AccessKeyEnv == b.AccessKeyEnv && a.SecretKeyEnv == b.SecretKeyEnv &&
		a.SessionTokenEnv == b.SessionTokenEnv
}

func equalOptions(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if vb, ok := b[k]; !ok || vb != v {
			return false
		}
	}
	return true
}
