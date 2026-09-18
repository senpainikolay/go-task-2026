package config_test

import (
	"testing"

	"recoTask/internal/config"
)

func TestNewConfig_RequiredFields(t *testing.T) {
	t.Setenv("ASANA_API_TOKEN", "test-token")
	t.Setenv("ASANA_API_DEFAULT_WORKSPACE", "123456")

	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.AsanaToken != "test-token" {
		t.Errorf("AsanaToken = %q, want %q", cfg.AsanaToken, "test-token")
	}
	if cfg.DefaultWorkspace != "123456" {
		t.Errorf("DefaultWorkspace = %q, want %q", cfg.DefaultWorkspace, "123456")
	}
}

func TestNewConfig_Defaults(t *testing.T) {
	t.Setenv("ASANA_API_TOKEN", "test-token")
	t.Setenv("ASANA_API_DEFAULT_WORKSPACE", "123456")

	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.LimiterMaxRetries != 3 {
		t.Errorf("LimiterMaxRetries = %d, want %d", cfg.LimiterMaxRetries, 3)
	}
	if cfg.MaxRequestsPerMinute != 120 {
		t.Errorf("MaxRequestsPerMinute = %d, want %d", cfg.MaxRequestsPerMinute, 120)
	}
	if cfg.PaginationLimit != 1 {
		t.Errorf("PaginationLimit = %d, want %d", cfg.PaginationLimit, 1)
	}
}

func TestNewConfig_OverridesDefaults(t *testing.T) {
	t.Setenv("ASANA_API_TOKEN", "test-token")
	t.Setenv("ASANA_API_DEFAULT_WORKSPACE", "123456")
	t.Setenv("ASANA_API_LIMITER_MAX_RETRIES", "5")
	t.Setenv("ASANA_API_MAXIMUM_REQUESTS_PER_MINUTE", "60")
	t.Setenv("ASANA_API_PAGINATION_LIMIT", "50")

	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.LimiterMaxRetries != 5 {
		t.Errorf("LimiterMaxRetries = %d, want %d", cfg.LimiterMaxRetries, 5)
	}
	if cfg.MaxRequestsPerMinute != 60 {
		t.Errorf("MaxRequestsPerMinute = %d, want %d", cfg.MaxRequestsPerMinute, 60)
	}
	if cfg.PaginationLimit != 50 {
		t.Errorf("PaginationLimit = %d, want %d", cfg.PaginationLimit, 50)
	}
}

func TestNewConfig_MissingRequiredField(t *testing.T) {
	t.Setenv("ASANA_API_TOKEN", "test-token")
	// ASANA_API_DEFAULT_WORKSPACE deliberately left unset.

	if _, err := config.NewConfig(); err == nil {
		t.Fatal("expected an error when a required env var is missing, got nil")
	}
}
