package healthcheck

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

type retryBackoff struct {
	counter   int
	threshold int
	retry     bool
}

func HealthCheck(logger *slog.Logger, done chan bool, ticker *time.Ticker, cfg Config, m *Metrics) {
	// TODO: With many endpoints we need to create some config mapping endpoint and parser together
	p := EnumToParser[cfg.Parser]

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
