package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	parsing "github.com/pattybardo/health_checker/internal/health_parsing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	HealthEndpointUrl     string
	CheckInterval         time.Duration
	ResponseTimeThreshold time.Duration
}

type metrics struct {
	responseTime  *prometheus.HistogramVec
	responseTotal *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		responseTime: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name: "healthcheck_http_response_time_seconds",
			Help: "Http response times in seconds",
		}, []string{"endpoint"}),
		responseTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "healthcheck_http_response_total",
			Help: "The total number of HTTP responses",
		}, []string{"http_status_code"}),
	}
	return m
}

// TODO: Split metrics functions out later
func recordMetrics(m *metrics, status string, elapsed time.Duration, url string) {
	m.responseTime.WithLabelValues(url).Observe(elapsed.Seconds())
	m.responseTotal.WithLabelValues(status).Inc()
}

func LoadConfig() (Config, error) {
	// TODO: Double check ParseDuration of emptyString is not 0, otherwise I need to check that
	checkInterval, err := time.ParseDuration(os.Getenv("CHECK_INTERVAL"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
	}

	responseTimeThreshold, err := time.ParseDuration(os.Getenv("RESPONSE_TIME_THRESHOLD"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid RESPONSE_TIME_THRESHOLD: %w", err)
	}

	healthEndpointUrl := os.Getenv("HEALTH_ENDPOINT_URL")

	if healthEndpointUrl == "" {
		return Config{}, fmt.Errorf("HEALTH_ENDPOINT_URL must not be empty")
	}

	config := Config{
		HealthEndpointUrl:     healthEndpointUrl,
		CheckInterval:         checkInterval,
		ResponseTimeThreshold: responseTimeThreshold,
	}

	return config, nil
}

func mockAlert(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
	// TODO: Mock slack message
}

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With()
	slog.SetDefault(logger)
	slog.Info("Starting health checker")

	cfg, err := LoadConfig()
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
		"responseThreshold", cfg.ResponseTimeThreshold.String(),
	)

	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	ticker := time.NewTicker(time.Duration(cfg.CheckInterval))
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		healthCheck(logger, done, ticker, cfg, m)
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	// TODO: Take inspiration from prom and have some endpoint status here maybe?
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

func healthCheck(logger *slog.Logger, done chan bool, ticker *time.Ticker, cfg Config, m *metrics) {
	// TODO: Hardcoded for now, maybe add default parser later.
	// TODO: With many endpoints this parser information needs to be chosen by I guess some sort of disocvery map.
	// i.e have the config loop naively check the endpoint and see if it maps to a parser we have setup
	ecp := &parsing.ElasticClusterParser{}
	for {
		select {
		case <-done:
			os.Exit(1)
		case <-ticker.C:
			start := time.Now()
			resp, err := http.Get(cfg.HealthEndpointUrl)
			elapsed := time.Since(start)
			if err != nil {
				logger.Error("HTTP error", "error", err.Error())
				mockAlert(logger, "Error getting endpoint", "error", err.Error())
				continue
			}
			healthChecklogger := logger.With(
				"service", ecp.ServiceName(),
				"status", resp.StatusCode,
				"responseTime", elapsed.String(),
			)

			if cfg.ResponseTimeThreshold < elapsed {
				healthChecklogger.Warn("Response time exceeded threshold")
			}

			if resp.StatusCode != http.StatusOK {
				mockAlert(healthChecklogger, "Non 200 status code")
			}

			parseResponse(healthChecklogger, ecp, resp)
			recordMetrics(m, resp.Status, elapsed, cfg.HealthEndpointUrl)
		}
	}
}

func parseResponse(logger *slog.Logger, parser parsing.HealthParser, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	err := resp.Body.Close()
	if err != nil {
		logger.Warn("Error closing the response body", "error", err)
	}
	result, err := parser.Parse(body)
	if err != nil {
		logger.Error("Parsing failure", "error", err)
		os.Exit(1)
	}
	switch result.Status {
	case parsing.StatusDegraded:
		logger.Warn("TODO: Retry transient")
	case parsing.StatusUnhealthy:
		logger.Error("TODO: Error log")
		mockAlert(logger, "Unhealthy service")
	default:
		logger.Info("Finished tick")
	}
}
