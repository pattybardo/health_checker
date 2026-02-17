package healthcheck_test

import (
	"strings"
	"testing"
	"time"

	"github.com/pattybardo/health_checker/healthcheck"
)

// TODO: Change invalid env vars to a typed error so we don't have to do string matching
func TestLoadConfig_InvalidCheckInterval(t *testing.T) {
	t.Setenv("CHECK_INTERVAL", "not-a-duration")
	t.Setenv("RESPONSE_TIME_THRESHOLD", "2s")
	t.Setenv("HEALTH_ENDPOINT_URL", "http://localhost:3000")
	t.Setenv("PARSER", "default")

	cfg, err := healthcheck.LoadConfig()
	if err == nil {
		t.Fatalf("LoadConfig() succeeded unexpectedly")
	}
	if !strings.Contains(err.Error(), "invalid CHECK_INTERVAL") {
		t.Fatalf("expected error to mention invalid CHECK_INTERVAL, got: %v", err)
	}
	if cfg != (healthcheck.Config{}) {
		t.Fatalf("Expected empty Config, got: %#v", cfg)
	}
}

func TestLoadConfig_CorrectParserDefault(t *testing.T) {
	t.Setenv("CHECK_INTERVAL", "5s")
	t.Setenv("RESPONSE_TIME_THRESHOLD", "2s")
	t.Setenv("HEALTH_ENDPOINT_URL", "http://localhost:3000")

	expectedParser := healthcheck.EnumDefaultParser

	cfg, err := healthcheck.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed unexpectedly, %v", err)
	}
	if expectedParser != cfg.Parser {
		t.Fatalf("LoadConfig() returned parser is different than expected, %#v", cfg.Parser)
	}

}

func TestLoadConfig_Correct(t *testing.T) {
	t.Setenv("CHECK_INTERVAL", "5s")
	t.Setenv("RESPONSE_TIME_THRESHOLD", "2s")
	t.Setenv("HEALTH_ENDPOINT_URL", "http://localhost:3000")
	t.Setenv("PARSER", "elasticsearch")

	expectedCfg := healthcheck.Config{
		HealthEndpointUrl:     "http://localhost:3000",
		CheckInterval:         5 * time.Second,
		ResponseTimeThreshold: 2 * time.Second,
		Parser:                healthcheck.EnumElasticsearchParser,
	}

	cfg, err := healthcheck.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed unexpectedly, %v", err)
	}
	if expectedCfg != cfg {
		t.Fatalf("LoadConfig() returned config is different than expected, %#v", cfg)
	}

}
