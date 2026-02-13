package main

import (
	"fmt"
	"io"
	"log"
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

func mockAlert() {
	fmt.Println("TODO: Mock Alert")
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v", err)
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	m := newMetrics(reg)

	ticker := time.NewTicker(time.Duration(cfg.CheckInterval))
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		time.Sleep(300000 * time.Second)
		done <- true
	}()
	go func() {
		for {
			select {
			case <-done:
				fmt.Println("Done!")
				return
			case t := <-ticker.C:
				start := time.Now()
				resp, err := http.Get(cfg.HealthEndpointUrl)
				elapsed := time.Since(start)
				if err != nil {
					panic(err)
				}

				if cfg.ResponseTimeThreshold < elapsed {
					fmt.Println("TODO: Warning")
				}

				if resp.StatusCode != http.StatusOK {
					mockAlert()
				}

				defer func() { _ = resp.Body.Close() }()
				body, _ := io.ReadAll(resp.Body)
				fmt.Println("get:\n", string(body))
				// TODO: Parse payload with parse interface and
				// use optional cluster health indicators
				// like non-green shard/cluster state to throw error log
				fmt.Println("Response time:", elapsed)
				recordMetrics(m, resp.Status, elapsed, cfg.HealthEndpointUrl)
				fmt.Println("Current time: ", t)
			}
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":2112", nil))

}
