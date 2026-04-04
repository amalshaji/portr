package config

import "testing"

func TestSetDefaultsAppliesDashboardPort(t *testing.T) {
	cfg := Config{}

	cfg.SetDefaults()

	if cfg.DashboardPort != DefaultDashboardPort {
		t.Fatalf("expected dashboard port %d, got %d", DefaultDashboardPort, cfg.DashboardPort)
	}
}

func TestGetDashboardDisableLabel(t *testing.T) {
	cfg := Config{
		DisableDashboard: true,
	}

	if got := cfg.GetDashboardDisableLabel(); got != "disabled via config" {
		t.Fatalf("expected disabled via config, got %q", got)
	}
}

func TestValidateRejectsInvalidDashboardPortWhenEnabled(t *testing.T) {
	cfg := Config{
		DashboardPort: 70000,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestValidateAllowsInvalidDashboardPortWhenDashboardDisabled(t *testing.T) {
	cfg := Config{
		DashboardPort:    70000,
		DisableDashboard: true,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestGetDashboardAddress(t *testing.T) {
	cfg := Config{
		DashboardPort: 8888,
	}

	if got := cfg.GetDashboardAddress(); got != "http://localhost:8888" {
		t.Fatalf("expected dashboard address http://localhost:8888, got %q", got)
	}

	cfg.DisableDashboard = true
	if got := cfg.GetDashboardAddress(); got != "" {
		t.Fatalf("expected disabled dashboard address to be empty, got %q", got)
	}
}
