package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-github/v84/github"
	"golang.org/x/oauth2"
)

type Config struct {
	GitHubToken       string
	Repos             []string // "owner/repo" format
	RemoteWriteURL    string
	RemoteWriteUser   string
	RemoteWritePass   string
	PollInterval      time.Duration
}

func loadConfig() Config {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN is required")
	}

	reposStr := os.Getenv("GITHUB_REPOS")
	if reposStr == "" {
		log.Fatal("GITHUB_REPOS is required")
	}
	repos := strings.Split(reposStr, ",")
	for i := range repos {
		repos[i] = strings.TrimSpace(repos[i])
	}

	rwURL := os.Getenv("REMOTE_WRITE_URL")
	if rwURL == "" {
		log.Fatal("REMOTE_WRITE_URL is required")
	}

	rwUser := os.Getenv("REMOTE_WRITE_USERNAME")
	if rwUser == "" {
		log.Fatal("REMOTE_WRITE_USERNAME is required")
	}

	rwPass := os.Getenv("REMOTE_WRITE_PASSWORD")
	if rwPass == "" {
		log.Fatal("REMOTE_WRITE_PASSWORD is required")
	}

	interval := 60 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("invalid POLL_INTERVAL %q: %v", v, err)
		}
		interval = d
	}

	return Config{
		GitHubToken:     token,
		Repos:           repos,
		RemoteWriteURL:  rwURL,
		RemoteWriteUser: rwUser,
		RemoteWritePass: rwPass,
		PollInterval:    interval,
	}
}

func main() {
	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.GitHubToken})
	ghClient := github.NewClient(oauth2.NewClient(ctx, ts))

	log.Printf("starting github PR exporter for %d repos, poll interval %s", len(cfg.Repos), cfg.PollInterval)

	run := func() {
		prs := collectPRs(ctx, ghClient, cfg.Repos)
		log.Printf("collected %d open PRs", len(prs))

		data := buildWriteRequest(prs)
		if err := pushMetrics(ctx, cfg, data); err != nil {
			log.Printf("error pushing metrics: %v", err)
		} else {
			log.Printf("pushed %d timeseries", len(prs))
		}
	}

	// Run immediately on startup.
	run()

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("shutting down")
			return
		case <-ticker.C:
			run()
		}
	}
}
