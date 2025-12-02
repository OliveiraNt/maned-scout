package cmd

import (
	"log"
	"os"

	httpserver "github.com/OliveiraNt/kdash/internal/adapters/http"
	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/infrastructure/kafka"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
)

func StartWeb() {
	configPath := os.Getenv("KDASH_CONFIG")
	if configPath == "" {
		configPath = "./clusters.yml" // TODO: find default location based on OS
	}

	// Initialize infrastructure layer
	factory := kafka.NewFactory()
	repo := repository.NewClusterRepository(configPath, factory)

	// Load initial configuration
	if err := repo.LoadFromFile(); err != nil {
		log.Printf("warning: failed to load config file: %v", err)
	}

	// Watch for configuration changes
	if err := repo.Watch(); err != nil {
		panic(err)
	}

	// Initialize application layer
	clusterService := application.NewClusterService(repo, factory)

	// Initialize presentation layer
	server := httpserver.New(clusterService, repo)

	port := os.Getenv("KDASH_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("HTTP UI running in :%s", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
