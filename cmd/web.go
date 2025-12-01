package cmd

import (
	"log"
	"os"

	httpserver "github.com/OliveiraNt/kdash/internal/adapters/http"
	"github.com/OliveiraNt/kdash/internal/registry"
)

func StartWeb() {
	configPath := os.Getenv("KDASH_CONFIG")
	if configPath == "" {
		configPath = "./clusters.yml"
	}

	reg := registry.New(configPath)
	if err := reg.LoadFromFile(configPath); err != nil {
		log.Printf("warning: failed to load config file: %v", err)
	}
	err := reg.Watch(configPath)
	if err != nil {
		panic(err)
	}

	server := httpserver.New(reg, configPath)
	log.Println("HTTP UI rodando em :8080")
	port := os.Getenv("KDASH_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	if err := server.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
