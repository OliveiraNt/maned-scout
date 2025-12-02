package cmd

import (
	"os"

	httpserver "github.com/OliveiraNt/kdash/internal/adapters/http"
	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/infrastructure/kafka"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"
)

func StartWeb() {
	// initialize structured logger
	registry.InitLogger()

	configPath := os.Getenv("KDASH_CONFIG")
	if configPath == "" {
		configPath = "./clusters.yml" // TODO: find default location based on OS
	}
	registry.Logger = registry.Logger.With("component", "cmd/web", "configPath", configPath)

	// Initialize infrastructure layer
	factory := kafka.NewFactory()
	repo := repository.NewClusterRepository(configPath, factory)
	registry.Logger.Info("initializing repository and kafka factory")

	// Load initial configuration
	if err := repo.LoadFromFile(); err != nil {
		registry.Logger.Warn("failed to load config file", "err", err)
	} else {
		registry.Logger.Info("configuration loaded")
	}
	// Watch for configuration changes
	if err := repo.Watch(); err != nil {
		registry.Logger.Error("failed to start config watcher", "err", err)
		panic(err)
	}

	// Initialize application layer
	clusterService := application.NewClusterService(repo, factory)
	registry.Logger.Info("application layer initialized")

	// Initialize presentation layer
	server := httpserver.New(clusterService, repo)
	port := os.Getenv("KDASH_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	registry.Logger.Info("HTTP UI starting", "port", port)
	if err := server.Run(":" + port); err != nil {
		registry.Logger.Fatal("HTTP UI terminated", "err", err)
	}
}
