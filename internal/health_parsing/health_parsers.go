package parsing

import (
	"encoding/json"
	"fmt"
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
