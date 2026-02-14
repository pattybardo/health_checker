package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

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

func mockAlert(msg string, args ...any) {
	slog.Error(msg, args...)
	// TODO: Mock slack message
}

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Starting health checker")

	cfg, err := LoadConfig()
	if err != nil {
		logger.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	ticker := time.NewTicker(time.Duration(cfg.CheckInterval))
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		healthCheck(done, ticker, cfg, m)
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":8989", nil))

}

func healthCheck(done chan bool, ticker *time.Ticker, cfg Config, m *metrics) {
	for {
		select {
		case <-done:
			os.Exit(1)
		case <-ticker.C:
			start := time.Now()
			resp, err := http.Get(cfg.HealthEndpointUrl)
			elapsed := time.Since(start)
			if err != nil {
				// TODO: Should I include the info in the requirements for this kind of error?
				slog.Error("HTTP error", "error", err.Error())
				mockAlert("Error getting endpoint", "endpoint", cfg.HealthEndpointUrl, "error", err.Error())
				continue
			}

			// req: Implement logging for all health checks, including timestamp,
			// endpoint, response time, and status.
			// Is there a way to set this in some default object that get's passed to make the logging cleaner?

			if cfg.ResponseTimeThreshold < elapsed {
				slog.Warn(
					"Response time exceeded threshold",
					"endpoint", cfg.HealthEndpointUrl,
					"responseTime", elapsed.String(),
					"responseThreshold", cfg.ResponseTimeThreshold.String(),
					"status", resp.StatusCode,
				)
			}

			if resp.StatusCode != http.StatusOK {
				mockAlert("Non 200 status code")
			}

			body, _ := io.ReadAll(resp.Body)
			err = resp.Body.Close()
			if err != nil {
				slog.Warn("Error closing the response body", "error", err)
			}
			slog.Info(string(body))
			// TODO: Parse payload with parse interface and
			// use optional cluster health indicators
			// like non-green shard/cluster state to throw error log
			slog.Info(
				"Finished tick",
				"endpoint", cfg.HealthEndpointUrl,
				"responseTime", elapsed.String(),
				"responseThreshold", cfg.ResponseTimeThreshold.String(),
				"status", resp.StatusCode,
			)
			recordMetrics(m, resp.Status, elapsed, cfg.HealthEndpointUrl)
		}
	}
}
