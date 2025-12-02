package repository

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/infrastructure/kafka"
	"github.com/fsnotify/fsnotify"
)

// ClusterRepository manages cluster configurations and their clients
type ClusterRepository struct {
	mu         sync.RWMutex
	clients    map[string]domain.KafkaClient
	configData config.FileConfig
	configPath string
	watcher    *fsnotify.Watcher
	factory    application.ClientFactory
}

// NewClusterRepository creates a new cluster repository
func NewClusterRepository(configPath string, factory application.ClientFactory) *ClusterRepository {
	return &ClusterRepository{
		clients:    make(map[string]domain.KafkaClient),
		configPath: configPath,
		factory:    factory,
	}
}

// LoadFromFile loads configuration from file
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

// Save persists a cluster configuration
func (r *ClusterRepository) Save(cfg config.ClusterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create or update client
	client, err := r.factory.CreateClient(cfg)
	if err != nil {
		return err
	}

	// If exists, close old client
	if old, ok := r.clients[cfg.Name]; ok {
		old.Close()
	}
	r.clients[cfg.Name] = client

	// Update in-memory config
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

	// Persist to file
	return r.writeToFile()
}

// Delete removes a cluster configuration by name
func (r *ClusterRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, ok := r.clients[name]
	if !ok {
		return application.ErrClusterNotFound
	}

	client.Close()
	delete(r.clients, name)

	// Remove from in-memory config
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

	// Persist to file
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

	const debounceDelay = 350 * time.Millisecond

	go func() {
		reload := func() {
			for i := 0; i < 10; i++ {
				if _, err := os.Stat(abs); err == nil {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			log.Printf("config file changed: %s", abs)
			if err := r.LoadFromFile(); err != nil {
				log.Printf("failed to reload config: %v", err)
			}
		}

		var timer *time.Timer
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Name != abs {
					continue
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove|fsnotify.Chmod) != 0 {
					if timer == nil {
						timer = time.AfterFunc(debounceDelay, reload)
					} else {
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						timer.Reset(debounceDelay)
					}
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("fsnotify error: %v", err)
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
			// Create new client
			client, err := r.factory.CreateClient(c)
			if err != nil {
				log.Printf("failed to create client for %s: %v", c.Name, err)
				continue
			}
			r.clients[c.Name] = client
			continue
		}

		// Check if config changed
		if kafkaClient, ok := cur.(*kafka.Client); ok {
			if !clusterConfigEqual(kafkaClient.GetConfig(), c) {
				cur.Close()
				client, err := r.factory.CreateClient(c)
				if err != nil {
					log.Printf("failed to recreate client for %s: %v", c.Name, err)
					continue
				}
				r.clients[c.Name] = client
			}
		}
	}

	// Remove clients not present in file
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
