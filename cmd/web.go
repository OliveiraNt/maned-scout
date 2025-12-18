package cmd

import (
	"os"

	httpserver "github.com/OliveiraNt/kdash/internal/adapters/http"
	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"
)

// StartWeb starts the HTTP server using already-initialized application and repository layers.
func StartWeb(clusterService *application.ClusterService, topicService *application.TopicService, repo *repository.ClusterRepository) {
	server := httpserver.New(clusterService, topicService, repo)
	port := os.Getenv("KDASH_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	registry.Logger.Info("HTTP UI starting", "port", port)
	if err := server.Run(":" + port); err != nil {
		registry.Logger.Fatal("HTTP UI terminated", "err", err)
	}
}
