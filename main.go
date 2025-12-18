package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/OliveiraNt/kdash/cmd"
	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/infrastructure/kafka"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"
	"github.com/joho/godotenv"
)

func findConfigPath() string {
	names := []string{"config.yml", "config.yaml"}
	candidates := []string{}

	for _, n := range names {
		candidates = append(candidates, "./"+n)
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(appdata, "kdash", n))
			}
		}
		if pd := os.Getenv("PROGRAMDATA"); pd != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(pd, "kdash", n))
			}
		}
		if home != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(home, "kdash", n))
			}
		}
	} else {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(xdg, "kdash", n))
			}
		}
		if home != "" {
			for _, n := range names {
				candidates = append(candidates, filepath.Join(home, ".config", "kdash", n))
				candidates = append(candidates, filepath.Join(home, ".kdash", n))
			}
		}
		for _, n := range names {
			candidates = append(candidates, filepath.Join("/etc", "kdash", n))
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
	initial := []byte("# kdash configuration\n")
	if err := os.WriteFile(createPath, initial, 0644); err == nil {
		return createPath
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return createPath
}

func main() {
	godotenv.Load()
	registry.InitLogger()

	configPath := os.Getenv("KDASH_CONFIG")
	if configPath == "" {
		configPath = findConfigPath()
	}

	factory := kafka.NewFactory()
	repo := repository.NewClusterRepository(configPath, factory)
	defer repo.Close()

	registry.Logger.Info("initializing repository and kafka factory")

	if err := repo.LoadFromFile(); err != nil {
		registry.Logger.Warn("failed to load config file", "err", err)
	} else {
		registry.Logger.Info("configuration loaded")
	}
	if err := repo.Watch(); err != nil {
		registry.Logger.Error("failed to start config watcher", "err", err)
		panic(err)
	}

	clusterService := application.NewClusterService(repo, factory)
	topicService := application.NewTopicService(clusterService, factory)
	registry.Logger.Info("application layer initialized")

	cmd.StartWeb(clusterService, topicService, repo)
}
