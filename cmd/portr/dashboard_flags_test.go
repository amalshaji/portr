package main

import (
	"testing"

	"github.com/amalshaji/portr/internal/client/config"
)

func TestApplyDashboardOptionsOverridesPortWithoutReEnablingDashboard(t *testing.T) {
	cfg := config.Config{
		DashboardPort:    config.DefaultDashboardPort,
		DisableDashboard: true,
	}

	err := applyDashboardOptions(&cfg, dashboardOptions{
		PortSet: true,
		Port:    8888,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DashboardPort != 8888 {
		t.Fatalf("expected dashboard port 8888, got %d", cfg.DashboardPort)
	}
	if !cfg.DisableDashboard {
		t.Fatal("expected dashboard to remain disabled")
	}
}

func TestApplyDashboardOptionsCanDisableDashboard(t *testing.T) {
	cfg := config.Config{
		DashboardPort: config.DefaultDashboardPort,
	}

	err := applyDashboardOptions(&cfg, dashboardOptions{
		DisabledSet: true,
		Disabled:    true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cfg.DisableDashboard {
		t.Fatal("expected dashboard to be disabled")
	}
	if cfg.DashboardDisableSource != config.DashboardDisableSourceCLI {
		t.Fatalf("expected disable source %q, got %q", config.DashboardDisableSourceCLI, cfg.DashboardDisableSource)
	}
}

func TestApplyDashboardOptionsRejectsConflictingFlags(t *testing.T) {
	cfg := config.Config{
		DashboardPort: config.DefaultDashboardPort,
	}

	err := applyDashboardOptions(&cfg, dashboardOptions{
		PortSet:     true,
		Port:        8888,
		DisabledSet: true,
		Disabled:    true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApplyDashboardOptionsRejectsInvalidPort(t *testing.T) {
	cfg := config.Config{
		DashboardPort: config.DefaultDashboardPort,
	}

	err := applyDashboardOptions(&cfg, dashboardOptions{
		PortSet: true,
		Port:    70000,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
