package config

import (
	"fmt"
	"runtime"
	"strings"
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
	GitHubAuthEnabled bool                       `json:"github_auth_enabled"`
	AutoSignupEnabled bool                       `json:"auto_signup_enabled"`
	AutoSignupDomains []AutoSignupDomainResponse `json:"auto_signup_domains"`
}

type AutoSignupDomainResponse struct {
	ID     uint                          `json:"id"`
	Domain string                        `json:"domain"`
	TeamID uint                          `json:"team_id"`
	Team   *InstanceSettingsTeamResponse `json:"team,omitempty"`
}

type InstanceSettingsTeamResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type UpdateInstanceSettingsInput struct {
	AutoSignupEnabled bool                    `json:"auto_signup_enabled"`
	AutoSignupDomains []AutoSignupDomainInput `json:"auto_signup_domains"`
}

type AutoSignupDomainInput struct {
	Domain string `json:"domain"`
	TeamID uint   `json:"team_id"`
}

func (h *Handler) GetInstanceSettings(c *fiber.Ctx) error {
	settings, err := models.GetOrCreateInstanceSettings(h.db)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load instance settings",
		})
	}

	domains, err := h.loadAutoSignupDomains()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load auto signup domains",
		})
	}

	return c.JSON(h.instanceSettingsResponse(settings, domains))
}

func (h *Handler) UpdateInstanceSettings(c *fiber.Ctx) error {
	var input UpdateInstanceSettingsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	autoSignupDomains, validationErr, err := h.buildAutoSignupDomainModels(input.AutoSignupDomains, input.AutoSignupEnabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate auto signup domains",
		})
	}
	if validationErr != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": validationErr,
		})
	}
	if input.AutoSignupEnabled {
		if !h.githubAuthEnabled() {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "GitHub authentication must be configured before enabling auto signup",
			})
		}
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		settings, err := models.GetOrCreateInstanceSettings(tx)
		if err != nil {
			return err
		}

		settings.AutoSignupEnabled = input.AutoSignupEnabled
		if err := tx.Save(settings).Error; err != nil {
			return err
		}

		if err := tx.Where("1 = 1").Delete(&models.AutoSignupDomain{}).Error; err != nil {
			return err
		}
		if len(autoSignupDomains) > 0 {
			if err := tx.Create(&autoSignupDomains).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update instance settings",
		})
	}

	settings, err := models.GetOrCreateInstanceSettings(h.db)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load instance settings",
		})
	}
	domains, err := h.loadAutoSignupDomains()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load auto signup domains",
		})
	}

	return c.JSON(h.instanceSettingsResponse(settings, domains))
}

func (h *Handler) githubAuthEnabled() bool {
	return h.config != nil && h.config.GithubClientID != "" && h.config.GithubSecret != ""
}

func (h *Handler) loadAutoSignupDomains() ([]models.AutoSignupDomain, error) {
	var domains []models.AutoSignupDomain
	if err := h.db.Preload("Team").Order("domain ASC").Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (h *Handler) buildAutoSignupDomainModels(input []AutoSignupDomainInput, requireDomains bool) ([]models.AutoSignupDomain, string, error) {
	seenDomains := make(map[string]uint, len(input))
	teamIDs := make(map[uint]struct{}, len(input))
	domains := make([]string, 0, len(input))
	autoSignupDomains := make([]models.AutoSignupDomain, 0, len(input))

	for _, item := range input {
		domain, ok := models.NormalizeAutoSignupDomain(item.Domain)
		if !ok {
			return nil, "Auto signup domain is invalid", nil
		}
		if item.TeamID == 0 {
			return nil, "Auto signup team is required for every domain", nil
		}

		if existingTeamID, ok := seenDomains[domain]; ok {
			if existingTeamID != item.TeamID {
				return nil, fmt.Sprintf("Domain %s is already configured for another team", domain), nil
			}
			return nil, fmt.Sprintf("Domain %s is configured more than once", domain), nil
		}

		seenDomains[domain] = item.TeamID
		teamIDs[item.TeamID] = struct{}{}
		domains = append(domains, domain)
		autoSignupDomains = append(autoSignupDomains, models.AutoSignupDomain{
			Domain: domain,
			TeamID: item.TeamID,
		})
	}

	if requireDomains && len(autoSignupDomains) == 0 {
		return nil, "At least one auto signup domain is required when auto signup is enabled", nil
	}

	if len(teamIDs) > 0 {
		ids := make([]uint, 0, len(teamIDs))
		for id := range teamIDs {
			ids = append(ids, id)
		}

		var count int64
		if err := h.db.Model(&models.Team{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
			return nil, "", err
		}
		if count != int64(len(ids)) {
			return nil, "Auto signup team does not exist", nil
		}
	}

	if len(domains) > 0 {
		var existing []models.AutoSignupDomain
		if err := h.db.Where("domain IN ?", domains).Find(&existing).Error; err != nil {
			return nil, "", err
		}
		for _, mapping := range existing {
			if seenDomains[mapping.Domain] != mapping.TeamID {
				return nil, fmt.Sprintf("Domain %s is already configured for another team", mapping.Domain), nil
			}
		}
	}

	return autoSignupDomains, "", nil
}

func (h *Handler) instanceSettingsResponse(settings *models.InstanceSettings, domains []models.AutoSignupDomain) InstanceSettingsResponse {
	autoSignupDomains := make([]AutoSignupDomainResponse, 0, len(domains))
	for _, domain := range domains {
		var team *InstanceSettingsTeamResponse
		if domain.Team.ID != 0 {
			team = &InstanceSettingsTeamResponse{
				ID:   domain.Team.ID,
				Name: domain.Team.Name,
				Slug: domain.Team.Slug,
			}
		}
		autoSignupDomains = append(autoSignupDomains, AutoSignupDomainResponse{
			ID:     domain.ID,
			Domain: domain.Domain,
			TeamID: domain.TeamID,
			Team:   team,
		})
	}

	return InstanceSettingsResponse{
		GitHubAuthEnabled: h.githubAuthEnabled(),
		AutoSignupEnabled: settings.AutoSignupEnabled,
		AutoSignupDomains: autoSignupDomains,
	}
}

func (h *Handler) DownloadConfig(c *fiber.Ctx) error {
	var input DownloadConfigInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var teamUser models.TeamUser
	err := h.db.Where("secret_key = ?", input.SecretKey).First(&teamUser).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid secret key",
		})
	}

	configContent := fmt.Sprintf(`server_url: %s
ssh_url: %s
secret_key: %s`, stripScheme(h.config.ServerURL), h.config.SshURL, teamUser.SecretKey)

	if h.config.SshHostKeyVerification {
		configContent += "\ninsecure_skip_host_key_verification: false"
	}

	configContent += `
tunnels:
  - name: portr
    subdomain: portr
    port: 4321`

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

func stripScheme(value string) string {
	return strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
}
