// Command server runs the effort dashboard backend.
//
//	server              run the backend
//	server -healthcheck exit 0 if the local backend answers /healthz (for container health checks)
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"linear-dashboard/backend/internal/app"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check the local backend's /healthz and exit")
	flag.Parse()
	if *healthcheck {
		os.Exit(checkHealth())
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := app.ConfigFromEnv()
	if err != nil {
		logger.Error("invalid configuration", "err", err)
		os.Exit(2)
	}
	if err := app.Run(context.Background(), cfg, logger); err != nil {
		logger.Error("backend stopped", "err", err)
		os.Exit(1)
	}
}

func checkHealth() int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8000/healthz")
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
