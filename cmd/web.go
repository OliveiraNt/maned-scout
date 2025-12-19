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
