package healthcheck

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	responseTime  *prometheus.HistogramVec
	responseTotal *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
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
func recordMetrics(m *Metrics, status string, elapsed time.Duration, url string) {
	m.responseTime.WithLabelValues(url).Observe(elapsed.Seconds())
	m.responseTotal.WithLabelValues(status).Inc()
}

func mockAlert(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
	// TODO: Mock slack message.
	// In reality I would add more context into a slack error message for what exactly is wrong so
	// we can get to the issue as fast as possible.

	// Using fmt.Println so logs are obvious visually for demonstration purposes.
	_, _ = fmt.Println("\nReaching out to @oncall in #our-favorite-channel...\n There seems to be issues...")
}
