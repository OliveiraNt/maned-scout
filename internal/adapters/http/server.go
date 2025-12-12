package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	clusterService *application.ClusterService
	repo           *repository.ClusterRepository
}

func New(clusterService *application.ClusterService, repo *repository.ClusterRepository) *Server {
	return &Server{
		clusterService: clusterService,
		repo:           repo,
	}
}

func (s *Server) Run(addr string) error {
	r := chi.NewRouter()
	// replace default logger with simple request log using our logger
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)
			registry.Logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", dur.String(),
			)
		})
	})

	// Statics Handler
	cacheDuration := 7 * 24 * time.Hour
	r.Handle("/static/*", http.StripPrefix("/static/", StaticWithCache("./internal/adapters/http/ui/static", cacheDuration)))

	// Web UI routes
	r.Get("/", s.uiHome)
	r.Get("/cluster/{name}", s.uiClusterDetail)
	r.Get("/cluster/{name}/topics", s.uiTopicsList)
	r.Get("/cluster/{name}/topics/{topic}", s.uiTopicDetail)

	// Cluster APIS
	r.Get("/api/clusters", s.apiListClusters)
	r.Post("/api/clusters", s.apiAddCluster)
	r.Put("/api/clusters/{name}", s.apiUpdateCluster)
	r.Delete("/api/clusters/{name}", s.apiDeleteCluster)

	// Topic APIs
	r.Get("/api/cluster/{name}/topics", s.apiListTopics)
	r.Get("/api/cluster/{name}/topics/{topic}", s.apiGetTopicDetail)
	r.Post("/api/cluster/{name}/topics", s.apiCreateTopic)
	r.Delete("/api/cluster/{name}/topics/{topic}", s.apiDeleteTopic)
	r.Put("/api/cluster/{name}/topics/{topic}/config", s.apiUpdateTopicConfig)
	r.Post("/api/cluster/{name}/topics/{topic}/partitions", s.apiIncreasePartitions)

	registry.Logger.Info("HTTP server listening", "addr", addr)
	return http.ListenAndServe(addr, r)
}

func StaticWithCache(dir string, maxAge time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		fullPath := filepath.Join(dir, filepath.Clean(path))

		// Verifica existência
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Header de cache
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(maxAge.Seconds())))

		http.ServeFile(w, r, fullPath)
	}
}
