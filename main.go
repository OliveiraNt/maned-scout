// Package main is the entry point for the Maned Scout application.
// Maned Scout is a Kafka cluster monitoring and management tool that provides
// a web interface for viewing and managing Kafka clusters, topics, and consumer groups.
// It supports multiple cluster configurations loaded from YAML files and monitors
// them in real-time.
package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/OliveiraNt/maned-scout/cmd"
	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/infrastructure/kafka"
	"github.com/OliveiraNt/maned-scout/internal/infrastructure/repository"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/joho/godotenv"
)

func findConfigPath() string {
	names := []string{"config.yml", "config.yaml"}
	candidates := make([]string, 0, 20)

	for _, n := range names {
		candidates = append(candidates, "./"+n)
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(appdata, "maned-scout", n))
			}
		}
		if pd := os.Getenv("PROGRAMDATA"); pd != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(pd, "maned-scout", n))
			}
		}
		if home != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(home, "maned-scout", n))
			}
		}
	} else {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(xdg, "maned-scout", n))
			}
		}
		if home != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(home, ".config", "maned-scout", n))
				candidates = append(candidates, filepath.Join(home, ".maned-scout", n))
			}
		}
		for _, n := range names {
			candidates = append(candidates, filepath.Join("/etc", "maned-scout", n))
		}
	}

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	createPath := "./config.yml"
	initial := []byte("# Maned Scout configuration\n")
	if err := os.WriteFile(createPath, initial, 0644); err == nil {
		return createPath
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return createPath
}

func main() {
	err := godotenv.Load()
	if err != nil {
		utils.Logger.Warn("failed to load .env file", "err", err)
	}
	utils.InitLogger()

	configPath := os.Getenv("MANED_SCOUT_CONFIG")
	if configPath == "" {
		configPath = findConfigPath()
	}

	factory := kafka.NewFactory()
	repo := repository.NewClusterRepository(configPath, factory)
	defer repo.Close()

	utils.Logger.Info("initializing repository and kafka factory")

	if err := repo.LoadFromFile(); err != nil {
		utils.Logger.Warn("failed to load config file", "err", err)
	} else {
		utils.Logger.Info("configuration loaded")
	}
	if err := repo.Watch(); err != nil {
		utils.Logger.Error("failed to start config watcher", "err", err)
		panic(err)
	}

	clusterService := application.NewClusterService(repo)
	utils.Logger.Info("application layer initialized")

	config.InitI18n()

	cmd.StartWeb(clusterService)
}
