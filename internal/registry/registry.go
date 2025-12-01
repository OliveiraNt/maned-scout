package registry

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/infra/kafka"

	"github.com/fsnotify/fsnotify"
)

// Registry holds active cluster clients and the in-memory config.
type Registry struct {
	mu         sync.RWMutex
	clients    map[string]*kafka.ClientWrapper
	config     config.FileConfig
	configPath string
	watcher    *fsnotify.Watcher
}

// New creates a registry; pass the path for YAML.
func New(configPath string) *Registry {
	return &Registry{
		clients:    make(map[string]*kafka.ClientWrapper),
		configPath: configPath,
	}
}

func (r *Registry) LoadFromFile(path string) error {
	cfg, err := config.ReadConfig(path)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()

	// reconcile: add/update/remove as per mix policy (3:c)
	return r.reconcile(cfg)
}

func (r *Registry) reconcile(cfg config.FileConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// mark existing
	existing := make(map[string]struct{})
	for _, c := range cfg.Clusters {
		existing[c.Name] = struct{}{}
		// add or update
		cur, ok := r.clients[c.Name]
		if !ok {
			// add
			w, err := kafka.NewKafkaClient(c)
			if err != nil {
				log.Printf("failed to create client for %s: %v", c.Name, err)
				continue
			}
			r.clients[c.Name] = w
			continue
		}
		// update: if config changed, replace client
		if !clusterConfigEqual(cur.Config, c) {
			cur.Close()
			w, err := kafka.NewKafkaClient(c)
			if err != nil {
				log.Printf("failed to recreate client for %s: %v", c.Name, err)
				continue
			}
			r.clients[c.Name] = w
		}
	}

	// remove clients not present in file
	for name, client := range r.clients {
		if _, ok := existing[name]; !ok {
			client.Close()
			delete(r.clients, name)
		}
	}
	return nil
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

// clusterConfigEqual compares relevant fields of cluster configs to decide
// whether an existing client needs to be recreated.
func clusterConfigEqual(a, b config.ClusterConfig) bool {
	if !equalStrings(a.Brokers, b.Brokers) {
		return false
	}
	if a.ClientID != b.ClientID {
		return false
	}
	if !equalTLS(a.TLS, b.TLS) {
		return false
	}
	if !equalSASL(a.SASL, b.SASL) {
		return false
	}
	if !equalAWS(a.AWS, b.AWS) {
		return false
	}
	if !equalOptions(a.Options, b.Options) {
		return false
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
	return a.Enabled == b.Enabled && a.CAFile == b.CAFile && a.CertFile == b.CertFile && a.KeyFile == b.KeyFile && a.InsecureSkipVerify == b.InsecureSkipVerify
}

func equalSASL(a, b *config.SASLConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Mechanism == b.Mechanism && a.Username == b.Username && a.Password == b.Password && a.UsernameEnv == b.UsernameEnv && a.PasswordEnv == b.PasswordEnv && a.ScramAlgorithm == b.ScramAlgorithm
}

func equalAWS(a, b *config.AWSConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.IAM == b.IAM && a.Region == b.Region && a.AccessKeyEnv == b.AccessKeyEnv && a.SecretKeyEnv == b.SecretKeyEnv && a.SessionTokenEnv == b.SessionTokenEnv
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

// Watch sets a fsnotify watcher on the file for hot reload.
func (r *Registry) Watch(path string) error {
	// ensure absolute path for watcher
	abs, err := filepath.Abs(path)
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
		for ev := range w.Events {
			// only react when the target file is written
			if ev.Name == abs && (ev.Op&fsnotify.Write == fsnotify.Write || ev.Op&fsnotify.Create == fsnotify.Create) {
				log.Printf("config file changed: %s", ev.Name)
				if err := r.LoadFromFile(path); err != nil {
					log.Printf("failed to reload config: %v", err)
				}
			}
		}
	}()
	return nil
}

// AddOrUpdateCluster used by API to add/update
func (r *Registry) AddOrUpdateCluster(c config.ClusterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// create client
	w, err := kafka.NewKafkaClient(c)
	if err != nil {
		return err
	}

	// if exists, close old
	if old, ok := r.clients[c.Name]; ok {
		old.Close()
	}
	r.clients[c.Name] = w

	// update in-memory config
	found := false
	for i := range r.config.Clusters {
		if r.config.Clusters[i].Name == c.Name {
			r.config.Clusters[i] = c
			found = true
			break
		}
	}
	if !found {
		r.config.Clusters = append(r.config.Clusters, c)
	}
	return nil
}

// DeleteCluster removes by name
func (r *Registry) DeleteCluster(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.clients[name]; ok {
		c.Close()
		delete(r.clients, name)
	} else {
		return errors.New("cluster not found")
	}

	// remove from in-memory config
	idx := -1
	for i := range r.config.Clusters {
		if r.config.Clusters[i].Name == name {
			idx = i
			break
		}
	}
	if idx >= 0 {
		r.config.Clusters = append(r.config.Clusters[:idx], r.config.Clusters[idx+1:]...)
	}
	return nil
}

// ListClusters returns configs (thread-safe)
func (r *Registry) ListClusters() []config.ClusterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]config.ClusterConfig, len(r.config.Clusters))
	copy(out, r.config.Clusters)
	return out
}

// GetClient returns wrapper for given cluster
func (r *Registry) GetClient(name string) (*kafka.ClientWrapper, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}

// WriteToFile persists current in-memory config to file (rewrite whole YAML)
func (r *Registry) WriteToFile(path string) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()
	// ensure dir exists
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	return config.WriteConfig(path, cfg)
}
