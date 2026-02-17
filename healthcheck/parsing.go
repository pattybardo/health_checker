package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"
)

type ParserEnum int

const (
	EnumDefaultParser ParserEnum = iota
	EnumElasticsearchParser
)

var EnumToParser = map[ParserEnum]HealthParser{
	EnumDefaultParser:       &DefaultParser{},
	EnumElasticsearchParser: &ElasticClusterParser{},
}

type Status int

const (
	StatusHealthy Status = iota
	StatusDegraded
	StatusUnhealthy
)

type Result struct {
	Status Status
	ID     string
}

// Create interface if we want to expand to other services later
type HealthParser interface {
	Parse(body []byte) (Result, error)
	ServiceName() string
}

// Default

type DefaultParser struct{}

func (dp *DefaultParser) Parse(body []byte) (Result, error) {
	return Result{
		// Default Parser could determine health from status code, but I am not sure
		// we want to bake the same business logic into both, or what we gain by doing so.
		// (Error on non 200 and also unhealthy)
		Status: StatusHealthy,
		ID:     "unknown",
	}, nil
}

func (dp *DefaultParser) ServiceName() string {
	return "unknown"
}

// Elastic

var elasticStringStatus = map[string]Status{
	"green":  StatusHealthy,
	"yellow": StatusDegraded,
	"red":    StatusUnhealthy,
}

type ElasticClusterParser struct{}

type elasticClusterHealth struct {
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
}

func (ep *ElasticClusterParser) Parse(body []byte) (Result, error) {
	var ech elasticClusterHealth
	if err := json.Unmarshal(body, &ech); err != nil {
		return Result{}, fmt.Errorf("parsing elasticsearch response: %w", err)
	}

	status := elasticStringStatus[ech.Status]

	return Result{
		Status: status,
		ID:     ech.ClusterName,
	}, nil
}

func (ep *ElasticClusterParser) ServiceName() string {
	return "elasticsearch"
}

// TODO: Refactor parseResponse return Result so we have a cleaner logic flow with no side-effects in this function
func parseResponse(logger *slog.Logger, parser HealthParser, resp *http.Response, retryCfg *retryBackoff) {
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
	case StatusDegraded:
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
	case StatusUnhealthy:
		mockAlert(logger, "Unhealthy service")
		retryCfg.retry = false
	default:
		logger.Info("Finished tick")
		retryCfg.retry = false
	}

}
