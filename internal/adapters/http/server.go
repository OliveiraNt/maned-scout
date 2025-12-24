package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/OliveiraNt/maned-scout/internal/adapters/http/mid"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server provides the HTTP API and Web UI endpoints for Maned Scout.
type Server struct {
	clusterService *application.ClusterService
	topicService   *application.TopicService
}

// New creates a new HTTP server instance.
func New(clusterService *application.ClusterService, topicService *application.TopicService) *Server {
	return &Server{
		clusterService: clusterService,
		topicService:   topicService,
	}
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	r := chi.NewRouter()
	r.Use(mid.I18n)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)
			utils.Logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", dur.String(),
			)
		})
	})

	cacheDuration := 7 * 24 * time.Hour
	r.Handle("/static/*", http.StripPrefix("/static/", StaticWithCache("./internal/adapters/http/ui/static", cacheDuration)))

	r.Get("/lang", ChangeLanguage)

	r.Get("/", s.uiHome)
	r.Get("/clusters/{clusterName}", s.uiClusterDetail)
	r.Get("/clusters/{clusterName}/topics", s.uiTopicsList)
	r.Get("/clusters/{clusterName}/topics/{topicName}", s.uiTopicDetail)
	r.Get("/clusters/{clusterName}/consumer-groups", s.uiConsumerGroupList)
	r.Get("/clusters/{clusterName}/consumer-groups/{consumerGroupName}", s.uiConsumerGroupDetail)

	r.Get("/api/clusters", s.apiListClusters)
	r.Post("/api/clusters", s.apiAddCluster)
	r.Put("/api/clusters/{clusterName}", s.apiUpdateCluster)
	r.Delete("/api/clusters/{clusterName}", s.apiDeleteCluster)

	r.Get("/api/clusters/{clusterName}/topics", s.apiListTopics)
	r.Get("/api/clusters/{clusterName}/topics/{topicName}", s.apiGetTopicDetail)
	r.Post("/api/clusters/{clusterName}/topics", s.apiCreateTopic)
	r.Delete("/api/clusters/{clusterName}/topics/{topicName}", s.apiDeleteTopic)
	r.Put("/api/clusters/{clusterName}/topics/{topicName}/config", s.apiUpdateTopicConfig)
	r.Post("/api/clusters/{clusterName}/topics/{topicName}/partitions", s.apiIncreasePartitions)
	r.Get("/api/clusters/{clusterName}/topics/{topicName}/ws-on", s.apiReadMessages)
	r.Get("/api/clusters/{clusterName}/topics/{topicName}/ws-off", s.apiStopMessages)
	r.Post("/api/clusters/{clusterName}/topics/{topicName}/messages", s.apiWriteMessage)
	r.Get("/api/clusters/{clusterName}/topics/{topicName}/ws", s.wsStreamTopic)
	r.Get("/api/clusters/{clusterName}/topics/{topicName}/consumer-groups", s.apiListTopicConsumerGroups)
	r.Get("/api/clusters/{clusterName}/consumer-groups", s.apiListConsumerGroup)

	utils.Logger.Info("HTTP server listening", "addr", addr)
	return http.ListenAndServe(addr, r)
}

// StaticWithCache serves static files from dir applying a public max-age cache header.
func StaticWithCache(dir string, maxAge time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		fullPath := filepath.Join(dir, filepath.Clean(path))
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(maxAge.Seconds())))

		http.ServeFile(w, r, fullPath)
	}
}

// ChangeLanguage changes the language preference via a query parameter and sets a cookie.
func ChangeLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   31536000,
	})

	// volta para a página anterior
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}

	http.Redirect(w, r, ref, http.StatusSeeOther)
}
