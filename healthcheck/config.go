package healthcheck

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

var EnvStringToParserEnum = map[string]ParserEnum{
	"default":       EnumDefaultParser,
	"elasticsearch": EnumElasticsearchParser,
}

type Config struct {
	HealthEndpointUrl     string
	CheckInterval         time.Duration
	ResponseTimeThreshold time.Duration
	Parser                ParserEnum
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

	parser, ok := EnvStringToParserEnum[parserFromEnv]
	if !ok {
		slog.Warn("Missing parser configuration. Setting default parser", "parser", parserFromEnv)
		parser = EnumDefaultParser
	}

	config := Config{
		HealthEndpointUrl:     healthEndpointUrl,
		CheckInterval:         checkInterval,
		ResponseTimeThreshold: responseTimeThreshold,
		Parser:                parser,
	}

	return config, nil
}
