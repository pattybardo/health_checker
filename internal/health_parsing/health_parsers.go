package parsing

import (
	"encoding/json"
	"fmt"
)

type Status int

const (
	StatusHealthy Status = iota
	StatusDegraded
	StatusUnhealthy
)

var elasticStringStatus = map[string]Status{
	"green":  StatusHealthy,
	"yellow": StatusDegraded,
	"red":    StatusUnhealthy,
}

type Result struct {
	Status Status
	ID     string
}

// Create interface if we want to expand to other services later
type HealthParser interface {
	Parse(body []byte) (Result, error)
	ServiceName() string
}

// Elastic

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
