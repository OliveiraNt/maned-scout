package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	pages "github.com/OliveiraNt/kdash/internal/adapters/http/ui/templates/pages"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/core"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	reg        *registry.Registry
	configPath string
}

func New(reg *registry.Registry, configPath string) *Server {
	return &Server{reg: reg, configPath: configPath}
}

func (s *Server) Run(addr string) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Web UI routes
	r.Get("/", s.uiHome)

	// Cluster APIS
	r.Get("/api/clusters", s.apiListClusters)
	r.Post("/api/clusters", s.apiAddCluster)
	r.Put("/api/clusters/{name}", s.apiUpdateCluster)
	r.Delete("/api/clusters/{name}", s.apiDeleteCluster)

	// Topic Apis
	r.Get("/api/cluster/{name}/topics", s.apiListTopics)

	return http.ListenAndServe(addr, r)
}

// uiHome renders the homepage listing clusters using templ + htmx + tailwind layout.
func (s *Server) uiHome(w http.ResponseWriter, r *http.Request) {
	cfgs := s.reg.ListClusters()
	// Map config clusters to core.Cluster for UI components.
	clusters := make([]core.Cluster, 0, len(cfgs))
	for _, c := range cfgs {
		clusters = append(clusters, core.Cluster{ID: c.Name, Name: c.Name, Brokers: c.Brokers})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ClusterList(clusters).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) apiListClusters(w http.ResponseWriter, r *http.Request) {
	clusters := s.reg.ListClusters()
	json.NewEncoder(w).Encode(clusters)
}

func (s *Server) apiAddCluster(w http.ResponseWriter, r *http.Request) {
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.Name == "" || len(c.Brokers) == 0 {
		http.Error(w, "invalid payload", 400)
		return
	}
	if err := s.reg.AddOrUpdateCluster(c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// persist
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(201)
}

func (s *Server) apiUpdateCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var c config.ClusterConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.Name == "" {
		c.Name = name
	}
	if err := s.reg.AddOrUpdateCluster(c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(204)
}

func (s *Server) apiDeleteCluster(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.reg.DeleteCluster(name); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if err := s.reg.WriteToFile(s.configPath); err != nil {
		log.Printf("failed to persist config: %v", err)
	}
	w.WriteHeader(204)
}

func (s *Server) apiListTopics(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	client, ok := s.reg.GetClient(name)
	if !ok {
		http.Error(w, "cluster not found", 404)
		return
	}
	topics, err := client.ListTopics()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(topics)
}
