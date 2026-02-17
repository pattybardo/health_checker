package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/pattybardo/health_checker/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With()
	slog.SetDefault(logger)
	slog.Info("Starting health checker")

	cfg, err := healthcheck.LoadConfig()
	if err != nil {
		logger.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	instance := os.Getenv("INSTANCE")
	if instance == "" {
		instance = "local"
	}

	logger = slog.Default().With(
		"instance", instance,
		"endpoint", cfg.HealthEndpointUrl,
	)

	reg := prometheus.NewRegistry()
	m := healthcheck.NewMetrics(reg)

	ticker := time.NewTicker(time.Duration(cfg.CheckInterval))
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		healthcheck.HealthCheck(logger, done, ticker, cfg, m)
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	healthyHandler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "Hey I seem to be healthy! :D")
	}
	http.HandleFunc("/health", healthyHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8989"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))

}
