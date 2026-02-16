package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	parsing "github.com/pattybardo/health_checker/internal/health_parsing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var envStringToParserEnum = map[string]parsing.ParserEnum{
	"default":       parsing.EnumDefaultParser,
	"elasticsearch": parsing.EnumElasticsearchParser,
}

type Config struct {
	HealthEndpointUrl     string
	CheckInterval         time.Duration
	ResponseTimeThreshold time.Duration
	Parser                parsing.ParserEnum
}

type metrics struct {
	responseTime  *prometheus.HistogramVec
	responseTotal *prometheus.CounterVec
}

type retryBackoff struct {
	counter   int
	threshold int
	retry     bool
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

	parserFromEnv := os.Getenv("PARSER")

	parser, ok := envStringToParserEnum[parserFromEnv]
	if !ok {
		slog.Warn("Missing parser configuration. Setting default parser", "parser", parserFromEnv)
		parser = parsing.EnumDefaultParser
	}

	config := Config{
		HealthEndpointUrl:     healthEndpointUrl,
		CheckInterval:         checkInterval,
		ResponseTimeThreshold: responseTimeThreshold,
		Parser:                parser,
	}

	return config, nil
}

func mockAlert(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
	// TODO: Mock slack message.
	// In reality I would add more context into a slack error message for what exactly is wrong so
	// we can get to the issue as fast as possible.

	// Using fmt.Println so logs are obvious visually for demonstration purposes.
	_, _ = fmt.Println("\nReaching out to @oncall in #our-favorite-channel...\n There seems to be issues...")
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
	// TODO: With many endpoints we need to create some config mapping endpoint and parser together
	p := parsing.EnumToParser[cfg.Parser]

	for {
		select {
		case <-done:
			os.Exit(1)
		case <-ticker.C:
			// TODO: The ticker sends a tick on the channel after every configured time. With the exponential
			// backoff, this could block the message, and then immediately be picked up. Not sure what the behavior
			// should be here, but one idea is to define the backoff outside the channel, and reset it inside once
			// there is a healthy check. Otherwise consecutive retries will not go through an exponential backoff loop
			retryCfg := retryBackoff{
				// TODO: Could make threshold configurable
				counter:   1,
				threshold: 4,
				retry:     true,
			}
			for retryCfg.retry {
				start := time.Now()
				resp, err := http.Get(cfg.HealthEndpointUrl)
				elapsed := time.Since(start)
				if err != nil {
					logger.Error("HTTP error", "error", err.Error())
					mockAlert(logger, "Error getting endpoint", "error", err.Error())
					retryCfg.retry = false
					continue
				}
				healthChecklogger := logger.With(
					"service", p.ServiceName(),
					"status", resp.StatusCode,
					"responseTime", elapsed.String(),
				)

				if cfg.ResponseTimeThreshold < elapsed {
					healthChecklogger.Warn("Response time exceeded threshold", "responseThreshold", cfg.ResponseTimeThreshold.String())
				}

				if resp.StatusCode != http.StatusOK {
					mockAlert(healthChecklogger, "Unhealthy endpoint")
				}

				parseResponse(healthChecklogger, p, resp, &retryCfg)
				// TODO: Add parsed info as prom metric somehow (elastic status maybe)
				recordMetrics(m, resp.Status, elapsed, cfg.HealthEndpointUrl)
			}
		}
	}
}

// TODO: Refactor parseResponse return parsing.Result so we have a cleaner logic flow with no side-effects in this function
func parseResponse(logger *slog.Logger, parser parsing.HealthParser, resp *http.Response, retryCfg *retryBackoff) {
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
		if retryCfg.counter >= retryCfg.threshold {
			retryCfg.retry = false
			logger.Error("Retry of expected transient failure did not succeed.")
			mockAlert(logger, "Service experiencing degradation over a long period of time.")
			return
		}
		sleepDuration := time.Duration(math.Pow(2, float64(retryCfg.counter)))
		logger.Warn("Retrying expected transient error with exponential backoff", "sleepSeconds", sleepDuration)
		time.Sleep(sleepDuration * time.Second)
		retryCfg.counter++
	case parsing.StatusUnhealthy:
		mockAlert(logger, "Unhealthy service")
		retryCfg.retry = false
	default:
		logger.Info("Finished tick")
		retryCfg.retry = false
	}

}
