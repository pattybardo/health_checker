package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	HealthEndpointUrl string
	// Play around with durations instead of ints, lets us more easily test timeouts
	// by setting ms timeouts.
	CheckInterval         time.Duration
	ResponseTimeThreshold time.Duration
}

type metrics struct {
	//responseTime   prometheus.Histogram
	responseTotal *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		responseTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "healthcheck_http_response_total",
			Help: "The total number of HTTP responses",
		}, []string{"http_status_code"}),
	}
	return m
}

func recordMetrics(m *metrics, status string) {
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
				recordMetrics(m, "200")
				fmt.Println("Current time: ", t)
			}
		}
	}()

	// reg.MustRegister(
	// 	collectors.NewGoCollector(),
	// 	collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	// )
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":2112", nil))

}

// type countHandler struct {
// 	mu sync.Mutex // guards n
// 	n  int
// }

// func Foo(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte(`{"bar":baz}`))
// }
