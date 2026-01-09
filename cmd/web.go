// Package cmd provides command implementations for the maned-scout application.
// It includes the StartWeb function which initializes and starts the HTTP server
// with the necessary application services and handles server configuration.
package cmd

import (
	"os"

	httpserver "github.com/OliveiraNt/maned-scout/internal/adapters/http"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/utils"
)

// StartWeb starts the HTTP server using already-initialized application and repository layers.
func StartWeb(clusterService *application.ClusterService) {
	topicService := application.NewTopicService(clusterService)
	server := httpserver.New(clusterService, topicService)
	port := os.Getenv("MANED_SCOUT_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	utils.Logger.Info("HTTP UI starting", "port", port)
	if err := server.Run(":" + port); err != nil {
		utils.Logger.Fatal("HTTP UI terminated", "err", err)
	}
}
