package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Custom messages for better type safety
type ErrorMsg struct {
	Error error
}

type AddTunnelMsg struct {
	Config       *config.Tunnel
	ClientConfig *config.ClientConfig
	Healthy      bool
}

type UpdateHealthMsg struct {
	Port    string
	Healthy bool
}

type AddLogMsg struct {
	Time   string
	Name   string
	Method string
	Status int
	URL    string
}

// Add new message type for debug logs
type AddDebugLogMsg struct {
	Time    string
	Level   string
	Message string
	Error   string
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff00")).
			MarginLeft(2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			MarginLeft(2).
			Bold(true)

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	healthyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00"))

	unhealthyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true)
)

type tickMsg time.Time

type tunnelStatus struct {
	config       *config.Tunnel
	clientConfig *config.ClientConfig
	healthy      bool
}

// Add debug table to model
type model struct {
	tunnels    map[string]*tunnelStatus
	table      table.Model
	debugTable table.Model // New debug table
	debug      bool        // Whether debug mode is enabled
	quitting   bool
	err        error
	lastUpdate time.Time
	selected   string
}

func New(debug bool) *tea.Program {
	// Regular table setup
	columns := []table.Column{
		{Title: "Time", Width: 12},
		{Title: "Tunnel", Width: 15},
		{Title: "Method", Width: 8},
		{Title: "Status", Width: 8},
		{Title: "URL", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	// Debug table setup
	debugColumns := []table.Column{
		{Title: "Time", Width: 12},
		{Title: "Level", Width: 8},
		{Title: "Message", Width: 50},
		{Title: "Error", Width: 30},
	}

	dt := table.New(
		table.WithColumns(debugColumns),
		table.WithFocused(false),
		table.WithHeight(10),
	)

	// Set styles for both tables
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(s)
	dt.SetStyles(s)

	return tea.NewProgram(
		model{
			tunnels:    make(map[string]*tunnelStatus),
			table:      t,
			debugTable: dt,
			debug:      debug,
		},
		tea.WithAltScreen(),
	)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "tab":
			// Cycle through tunnels
			var ports []string
			for port := range m.tunnels {
				ports = append(ports, port)
			}
			if len(ports) > 0 {
				for i, port := range ports {
					if port == m.selected {
						m.selected = ports[(i+1)%len(ports)]
						break
					}
				}
			}
		}

	case ErrorMsg:
		m.err = msg.Error
		m.quitting = true
		return m, tea.Quit

	case AddTunnelMsg:
		port := fmt.Sprintf("%d", msg.Config.Port)
		m.tunnels[port] = &tunnelStatus{
			config:       msg.Config,
			clientConfig: msg.ClientConfig,
			healthy:      msg.Healthy,
		}
		if m.selected == "" {
			m.selected = port
		}

	case UpdateHealthMsg:
		if tunnel, exists := m.tunnels[msg.Port]; exists {
			tunnel.healthy = msg.Healthy
		}

	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width - 4)
		m.debugTable.SetWidth(msg.Width - 4)
		return m, nil

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tickCmd()

	case AddLogMsg:
		m.AddLog(msg)
		return m, nil

	case AddDebugLogMsg:
		m.AddDebugLog(msg)
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
		}
		return "Goodbye!\n"
	}

	var s string

	s += titleStyle.Render("🌍 Portr Tunnel Dashboard") + "\n"
	s += subtitleStyle.Render(fmt.Sprintf("Active Tunnels: %d", len(m.tunnels))) + "\n"
	s += subtitleStyle.Render("Local Dashboard: http://localhost:7777") + "\n\n"

	if len(m.tunnels) == 0 {
		s += subtitleStyle.Render("Waiting for tunnels to connect...") + "\n"
		return s
	}

	// Create a sorted slice of tunnels
	type sortableTunnel struct {
		port   string
		tunnel *tunnelStatus
	}
	sortedTunnels := make([]sortableTunnel, 0, len(m.tunnels))
	for port, tunnel := range m.tunnels {
		sortedTunnels = append(sortedTunnels, sortableTunnel{port, tunnel})
	}
	// Sort by subdomain
	sort.Slice(sortedTunnels, func(i, j int) bool {
		subdomainI := sortedTunnels[i].tunnel.config.Subdomain
		subdomainJ := sortedTunnels[j].tunnel.config.Subdomain
		// If subdomains are empty, sort by name
		if subdomainI == "" && subdomainJ == "" {
			return sortedTunnels[i].tunnel.config.Name < sortedTunnels[j].tunnel.config.Name
		}
		// Empty subdomains go last
		if subdomainI == "" {
			return false
		}
		if subdomainJ == "" {
			return true
		}
		return subdomainI < subdomainJ
	})

	// Show tunnel statuses in sorted order
	for _, st := range sortedTunnels {
		tunnel := st.tunnel
		var tunnelStyle lipgloss.Style
		var statusText string

		if tunnel.healthy {
			tunnelStyle = healthyStyle
			statusText = "🟢 Healthy"
		} else {
			tunnelStyle = unhealthyStyle
			statusText = "🟡 Reconnecting"
		}

		tunnelInfo := fmt.Sprintf("%s (%s:%d → %s) [%s] %s",
			tunnel.config.Name,
			tunnel.config.Host,
			tunnel.config.Port,
			tunnel.clientConfig.GetTunnelAddr(),
			tunnel.config.Subdomain,
			statusText,
		)
		s += tunnelStyle.Render(tunnelInfo) + "\n"
	}
	s += "\n"

	// Just render the table - no need to query DB
	s += tableStyle.Render(m.table.View()) + "\n"

	// Show debug table if debug mode is enabled
	if m.debug {
		s += "\n" + titleStyle.Render("🔍 Debug Logs") + "\n"
		s += tableStyle.Render(m.debugTable.View()) + "\n"
	}

	// Help and status
	s += "\n" + subtitleStyle.Render("Ctrl+C: Quit") + "\n"
	s += subtitleStyle.Render(fmt.Sprintf("Last updated: %s", m.lastUpdate.Format("15:04:05"))) + "\n"

	return s
}

func (m *model) AddLog(msg AddLogMsg) {
	rows := []table.Row{{
		msg.Time,
		msg.Name,
		msg.Method,
		fmt.Sprintf("%d", msg.Status),
		msg.URL,
	}}

	// Get existing rows and prepend new row
	existingRows := m.table.Rows()
	if len(existingRows) > 48 { // Keep last 49 to make room for new row
		existingRows = existingRows[:48]
	}

	// Combine new row with existing rows
	allRows := append(rows, existingRows...)
	m.table.SetRows(allRows)
}

// Add method to handle debug logs
func (m *model) AddDebugLog(msg AddDebugLogMsg) {
	if !m.debug {
		return
	}

	rows := []table.Row{{
		msg.Time,
		msg.Level,
		msg.Message,
		msg.Error,
	}}

	// Get existing rows and prepend new row
	existingRows := m.debugTable.Rows()
	if len(existingRows) > 48 {
		existingRows = existingRows[:48]
	}

	// Combine new row with existing rows
	allRows := append(rows, existingRows...)
	m.debugTable.SetRows(allRows)
}
