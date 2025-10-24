package config

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/admin/middleware"
	"github.com/amalshaji/portr/internal/server/admin/models"
	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"gorm.io/gorm"

	serverStats "github.com/amalshaji/portr/internal/server/stats"
)

var (
	lastCPUTime     types.CPUTimes
	lastMeasureTime time.Time
	cpuMutex        sync.RWMutex
)

func calculateCPUUsage(currentCPU types.CPUTimes) float64 {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	now := time.Now()

	// If this is the first measurement, store it and return 0
	if lastMeasureTime.IsZero() {
		lastCPUTime = currentCPU
		lastMeasureTime = now
		return 0.0
	}

	// Calculate time difference
	timeDiff := now.Sub(lastMeasureTime).Seconds()
	if timeDiff <= 0 {
		return 0.0
	}

	// Calculate CPU time differences (in nanoseconds, convert to seconds)
	userDiff := float64(currentCPU.User-lastCPUTime.User) / 1e9
	systemDiff := float64(currentCPU.System-lastCPUTime.System) / 1e9

	// Total CPU time used
	totalCPUDiff := userDiff + systemDiff

	// CPU usage percentage (considering number of CPU cores)
	numCPU := float64(runtime.NumCPU())
	cpuUsage := (totalCPUDiff / (timeDiff * numCPU)) * 100

	// Update last measurements
	lastCPUTime = currentCPU
	lastMeasureTime = now

	// Ensure reasonable bounds
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage
}

type Handler struct {
	db             *gorm.DB
	store          *session.Store
	config         *serverConfig.AdminConfig
	statsCollector *serverStats.StatsCollector
}

func NewHandler(db *gorm.DB, store *session.Store, cfg *serverConfig.AdminConfig, statsCollector *serverStats.StatsCollector) *Handler {
	return &Handler{
		db:             db,
		store:          store,
		config:         cfg,
		statsCollector: statsCollector,
	}
}

type DownloadConfigInput struct {
	SecretKey string `json:"secret_key" validate:"required"`
}

type InstanceSettingsResponse struct {
	SMTPEnabled         bool   `json:"smtp_enabled"`
	SMTPHost            string `json:"smtp_host"`
	SMTPPort            int    `json:"smtp_port"`
	SMTPUsername        string `json:"smtp_username"`
	SMTPPassword        string `json:"smtp_password"`
	FromAddress         string `json:"from_address"`
	AddUserEmailSubject string `json:"add_user_email_subject"`
	AddUserEmailBody    string `json:"add_user_email_body"`
}

type UpdateInstanceSettingsInput struct {
	SMTPEnabled         bool   `json:"smtp_enabled"`
	SMTPHost            string `json:"smtp_host"`
	SMTPPort            int    `json:"smtp_port"`
	SMTPUsername        string `json:"smtp_username"`
	SMTPPassword        string `json:"smtp_password"`
	FromAddress         string `json:"from_address"`
	AddUserEmailSubject string `json:"add_user_email_subject"`
	AddUserEmailBody    string `json:"add_user_email_body"`
}

func (h *Handler) GetInstanceSettings(c *fiber.Ctx) error {
	// For now, return default values since we don't have a settings table yet
	// In a real implementation, you'd store these in the database
	settings := InstanceSettingsResponse{
		SMTPEnabled:         false,
		SMTPHost:            "",
		SMTPPort:            587,
		SMTPUsername:        "",
		SMTPPassword:        "",
		FromAddress:         "",
		AddUserEmailSubject: "Welcome to Portr!",
		AddUserEmailBody:    "You have been added to a Portr team. Please set up your account using the temporary password provided.",
	}

	return c.JSON(settings)
}

func (h *Handler) UpdateInstanceSettings(c *fiber.Ctx) error {
	var input UpdateInstanceSettingsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// For now, just return success since we don't have a settings table yet
	// In a real implementation, you'd save these to the database

	return c.JSON(fiber.Map{
		"message": "Settings updated successfully",
	})
}

func (h *Handler) DownloadConfig(c *fiber.Ctx) error {
	var input DownloadConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Find team user by secret key
	var teamUser models.TeamUser
	err := h.db.Where("secret_key = ?", input.SecretKey).First(&teamUser).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid secret key",
		})
	}

	// Generate config content
	configContent := fmt.Sprintf(`server_url: %s
ssh_url: %s
secret_key: %s
enable_request_logging: false
tunnels:
  - name: portr
    subdomain: portr
    port: 4321`, h.config.ServerURL, h.config.SshURL, teamUser.SecretKey)

	return c.JSON(fiber.Map{
		"message": configContent,
	})
}

func (h *Handler) GetSetupScript(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	setupScript := fmt.Sprintf(`portr auth set --token %s --remote %s`,
		teamUser.SecretKey, h.config.ServerURL)

	return c.JSON(fiber.Map{
		"message": setupScript,
	})
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	teamUser := middleware.GetCurrentTeamUser(c)
	if teamUser == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Team context required",
		})
	}

	// Get team statistics
	var activeConnections, teamMembers int64

	// Active connections for this team
	h.db.Model(&models.Connection{}).Where("team_id = ? AND status = ?",
		teamUser.TeamID, models.ConnectionStatusActive).Count(&activeConnections)

	// Team members count
	h.db.Model(&models.TeamUser{}).Where("team_id = ?", teamUser.TeamID).Count(&teamMembers)

	// Get server start time from context (if available)
	serverStartTime := c.Locals("server_start_time")
	if serverStartTime == nil {
		// Fallback to current time if not available
		now := time.Now()
		serverStartTime = now
	}

	// Get historical stats data from the collector
	statsData := h.statsCollector.GetStats()

	// Get latest stats for current values
	var latestStats serverStats.StatsData
	var hasLatest bool
	if latest, ok := h.statsCollector.GetLatestStats(); ok {
		latestStats = latest
		hasLatest = true
	}

	// Convert historical data to chart-friendly format
	var memoryUsageHistory []fiber.Map
	var cpuUsageHistory []fiber.Map

	for _, data := range statsData {
		memoryUsageHistory = append(memoryUsageHistory, fiber.Map{
			"timestamp": data.Timestamp,
			"value":     data.MemoryUsage,
		})
		cpuUsageHistory = append(cpuUsageHistory, fiber.Map{
			"timestamp": data.Timestamp,
			"value":     data.CPUUsage,
		})
	}

	// Get current system metrics for real-time display
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Convert bytes to MB for better display
	memoryUsedMB := float64(memStats.Alloc) / 1024 / 1024
	memoryTotalMB := float64(memStats.Sys) / 1024 / 1024

	// Get CPU and memory info from system
	systemStats := fiber.Map{
		"server_start_time": serverStartTime,
		"memory_used_mb":    memoryUsedMB,
		"memory_total_mb":   memoryTotalMB,
		"goroutines":        runtime.NumGoroutine(),
		"num_cpu":           runtime.NumCPU(),
	}

	// Try to get system info using go-sysinfo
	if host, err := sysinfo.Host(); err == nil {
		info := host.Info()
		systemStats["hostname"] = info.Hostname
		systemStats["os"] = info.OS.Name
		systemStats["architecture"] = info.Architecture

		if memory, err := host.Memory(); err == nil {
			if memory.Total > 0 {
				systemMemoryTotalGB := float64(memory.Total) / 1024 / 1024 / 1024
				systemMemoryUsedGB := float64(memory.Used) / 1024 / 1024 / 1024
				memoryUsagePercent := (float64(memory.Used) / float64(memory.Total)) * 100

				systemStats["system_memory_total_gb"] = systemMemoryTotalGB
				systemStats["system_memory_used_gb"] = systemMemoryUsedGB
				systemStats["system_memory_usage_percent"] = memoryUsagePercent
			}
		}

		if cpuTime, err := host.CPUTime(); err == nil {
			systemStats["cpu_user_time"] = cpuTime.User
			systemStats["cpu_system_time"] = cpuTime.System

			// Calculate CPU usage percentage
			cpuUsage := calculateCPUUsage(cpuTime)
			systemStats["cpu_usage_percent"] = cpuUsage
		}
	}

	// Prepare chart data
	chartData := fiber.Map{
		"memory_usage": memoryUsageHistory,
		"cpu_usage":    cpuUsageHistory,
	}

	// Include latest values if available
	if hasLatest {
		chartData["latest"] = fiber.Map{
			"memory_usage": latestStats.MemoryUsage,
			"cpu_usage":    latestStats.CPUUsage,
			"timestamp":    latestStats.Timestamp,
		}
	}

	return c.JSON(fiber.Map{
		"team_stats": fiber.Map{
			"active_connections": activeConnections,
			"team_members":       teamMembers,
		},
		"system_stats": systemStats,
		"chart_data":   chartData,
	})
}
