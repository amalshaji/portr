package tui

import (
	"fmt"
	"sort"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Custom messages for better type safety
type ErrorMsg struct {
	Error error
}

type QuitMsg struct{}

type AddTunnelMsg struct {
	Config       *config.Tunnel
	ClientConfig *config.ClientConfig
	Healthy      bool
}

type UpdateHealthMsg struct {
	Port    string
	Healthy bool
}

// UpdateConnCountMsg adjusts the active connection count for a tunnel (Delta can be +1 or -1)
type UpdateConnCountMsg struct {
	Port  string
	Delta int
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

// Pure #00ff00 and ANSI yellow are unreadable on light backgrounds, so both
// resolve per terminal theme; red and gray read fine on either.
var (
	green  = lipgloss.AdaptiveColor{Light: "#1a7f37", Dark: "#00ff00"}
	yellow = lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "yellow"}
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(green).
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
			Foreground(green)

	unhealthyStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	// red style for fully unhealthy state (no active connections)
	redStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Bold(true)
)

type tickMsg time.Time

type tunnelStatus struct {
	config       *config.Tunnel
	clientConfig *config.ClientConfig
	healthy      bool
	active       int
	poolSize     int
}

// Add debug table to model
type model struct {
	tunnels                map[string]*tunnelStatus
	table                  table.Model
	debugTable             table.Model // New debug table
	debug                  bool        // Whether debug mode is enabled
	quitting               bool
	err                    error
	lastUpdate             time.Time
	selected               string
	width                  int
	height                 int
	dashboardURL           string
	dashboardDisabledLabel string
	qrEnabled              bool
	showQR                 bool
	qrPanel                string
}

// applyLayout resizes the tables for the given terminal size. The QR panel is
// rebuilt here because it competes with the request table for vertical space.
func (m *model) applyLayout(width, height int) {
	m.width = width
	m.height = height
	m.qrPanel = m.buildQRPanel()

	// Calculate dynamic widths based on terminal size
	totalWidth := max(width-6, 30) // Account for borders and terminal edges

	// Adjust table heights based on terminal height
	availableHeight := height - 15 // Account for headers and other UI elements
	if m.qrPanel != "" {
		availableHeight -= lipgloss.Height(m.qrPanel) + 1
	}
	availableHeight = max(availableHeight, 8)

	if m.debug {
		// Prioritize request logs when debug is enabled.
		mainTableHeight := max((availableHeight*2)/3, 8)
		debugTableHeight := max(availableHeight-mainTableHeight-1, 4)
		m.table.SetHeight(mainTableHeight)
		m.debugTable.SetHeight(debugTableHeight)
	} else {
		// Use nearly all available space for request logs when debug table is hidden.
		m.table.SetHeight(availableHeight)
	}

	// Adjust URL column width to fill remaining space
	timeWidth := 12
	tunnelWidth := 15
	methodWidth := 8
	statusWidth := 8
	mainCellPadding := 2 * 5 // table default style adds left/right padding to each cell
	urlWidth := totalWidth - (timeWidth + tunnelWidth + methodWidth + statusWidth + mainCellPadding + 1)

	urlWidth = max(urlWidth, 10)

	// Update main table columns
	cols := []table.Column{
		{Title: "Time", Width: timeWidth},
		{Title: "Tunnel", Width: tunnelWidth},
		{Title: "Method", Width: methodWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "URL", Width: urlWidth},
	}
	m.table.SetColumns(cols)

	// Update debug table columns if debug is enabled
	if m.debug {
		debugCellPadding := 2 * 4 // four columns with left/right padding
		debugContentWidth := totalWidth - (timeWidth + methodWidth + debugCellPadding + 1)
		debugContentWidth = max(debugContentWidth, 24)

		messageWidth := max(debugContentWidth/2, 12)
		errorWidth := max(debugContentWidth-messageWidth, 12)

		debugCols := []table.Column{
			{Title: "Time", Width: timeWidth},
			{Title: "Level", Width: methodWidth},
			{Title: "Message", Width: messageWidth},
			{Title: "Error", Width: errorWidth},
		}
		m.debugTable.SetColumns(debugCols)
	}

	m.table.SetWidth(totalWidth)
	m.debugTable.SetWidth(totalWidth)
}

func tunnelKey(tunnelConfig *config.Tunnel) string {
	if tunnelConfig == nil {
		return ""
	}
	return tunnelConfig.StatusKey()
}

func New(debug bool, dashboardURL string, dashboardDisabledLabel string, enableQRCode bool) *tea.Program {
	// Resolve the terminal background now, while stdin is still ours; once
	// bubbletea owns input the OSC query times out and defaults to dark.
	lipgloss.HasDarkBackground()

	// Initial default widths
	const (
		timeWidth   = 12
		tunnelWidth = 15
		methodWidth = 8
		statusWidth = 8
		urlWidth    = 50
	)

	// Regular table setup with minimum widths
	columns := []table.Column{
		{Title: "Time", Width: timeWidth},
		{Title: "Tunnel", Width: tunnelWidth},
		{Title: "Method", Width: methodWidth},
		{Title: "Status", Width: statusWidth},
		{Title: "URL", Width: urlWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(14), // Larger default height
	)

	// Debug table setup with minimum widths
	debugColumns := []table.Column{
		{Title: "Time", Width: timeWidth},
		{Title: "Level", Width: methodWidth},
		{Title: "Message", Width: 30},
		{Title: "Error", Width: 20},
	}

	dt := table.New(
		table.WithColumns(debugColumns),
		table.WithFocused(false),
		table.WithHeight(6),
	)

	// Set styles for both tables
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)

	t.SetStyles(s)
	dt.SetStyles(s)

	return tea.NewProgram(
		model{
			tunnels:                make(map[string]*tunnelStatus),
			table:                  t,
			debugTable:             dt,
			debug:                  debug,
			width:                  80,
			dashboardURL:           dashboardURL,
			dashboardDisabledLabel: dashboardDisabledLabel,
			qrEnabled:              enableQRCode,
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
		if m.err != nil {
			// Any key press when there's an error will quit
			m.quitting = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			if m.qrEnabled {
				m.showQR = !m.showQR
				m.applyLayout(m.width, m.height)
			}
			// Swallow the key either way so the table never scrolls on it.
			return m, nil
		}

	case ErrorMsg:
		m.err = msg.Error
		// Don't quit immediately, let the user see the error
		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case AddTunnelMsg:
		key := tunnelKey(msg.Config)
		if existing, ok := m.tunnels[key]; ok {
			// Update pool size if provided; do not change active here
			if msg.ClientConfig != nil {
				existing.poolSize = max(existing.poolSize, msg.ClientConfig.Tunnel.PoolSize)
			}
		} else {
			ps := 1
			if msg.ClientConfig != nil {
				ps = msg.ClientConfig.Tunnel.PoolSize
			}
			m.tunnels[key] = &tunnelStatus{
				config:       msg.Config,
				clientConfig: msg.ClientConfig,
				healthy:      msg.Healthy,
				active:       0,
				poolSize:     ps,
			}
		}
		if m.selected == "" {
			m.selected = key
		}
		if m.showQR {
			// The panel may have been unavailable before this tunnel connected.
			m.applyLayout(m.width, m.height)
		}

	case UpdateHealthMsg:
		if tunnel, exists := m.tunnels[msg.Port]; exists {
			tunnel.healthy = msg.Healthy
		}

	case UpdateConnCountMsg:
		if tunnel, exists := m.tunnels[msg.Port]; exists {
			tunnel.active += msg.Delta
			if tunnel.active < 0 {
				tunnel.active = 0
			}
			tunnel.active = min(tunnel.active, max(1, tunnel.poolSize))
		}

	case tea.WindowSizeMsg:
		m.applyLayout(msg.Width, msg.Height)
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
			return "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		}
		return "Goodbye!\n"
	}

	if m.err != nil {
		return "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n" +
			subtitleStyle.Render("Press any key to exit...") + "\n"
	}

	var s string

	s += titleStyle.Render("🌍 Portr Tunnel Dashboard") + "\n"
	s += subtitleStyle.Render(fmt.Sprintf("Active Tunnels: %d", len(m.tunnels))) + "\n"
	if m.dashboardURL == "" {
		label := "disabled"
		if m.dashboardDisabledLabel != "" {
			label = m.dashboardDisabledLabel
		}
		s += subtitleStyle.Render("Local Dashboard: "+label) + "\n\n"
	} else {
		s += subtitleStyle.Render("Local Dashboard: "+m.dashboardURL) + "\n\n"
	}

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
	lineWidth := max(m.width-4, 20)
	for _, st := range sortedTunnels {
		tunnel := st.tunnel
		var tunnelStyle lipgloss.Style
		var statusText string

		// Health coloring:
		// - No active connections => red (unhealthy)
		// - Active < poolSize => yellow (partially unhealthy)
		// - Active >= poolSize => green (healthy)
		if tunnel.active == 0 {
			tunnelStyle = redStyle
			statusText = "🔴 Unhealthy (0/" + fmt.Sprint(max(1, tunnel.poolSize)) + ")"
		} else if tunnel.active < max(1, tunnel.poolSize) {
			tunnelStyle = unhealthyStyle
			statusText = "🟡 Partial (" + fmt.Sprint(tunnel.active) + "/" + fmt.Sprint(max(1, tunnel.poolSize)) + ")"
		} else {
			tunnelStyle = healthyStyle
			statusText = "🟢 Healthy (" + fmt.Sprint(tunnel.active) + "/" + fmt.Sprint(max(1, tunnel.poolSize)) + ")"
		}

		if tunnel.config.ResolvedBasicAuth() != "" {
			statusText = "🔒 " + statusText
		}

		tunnelName := tunnel.config.DisplayName()
		tunnelAddr := ""
		if tunnel.clientConfig != nil {
			tunnelAddr = tunnel.clientConfig.GetTunnelAddr()
		}

		var tunnelInfo string
		if tunnel.config.Type == constants.Stub || tunnel.config.Type == constants.Static {
			// The served directory is absolute and would push the public URL off
			// the end of this width-clamped line; it is printed at startup instead.
			tunnelInfo = fmt.Sprintf("%s (%s → %s) [%s] %s",
				tunnelName,
				tunnel.config.Type,
				tunnelAddr,
				tunnel.config.Subdomain,
				statusText,
			)
		} else {
			tunnelInfo = fmt.Sprintf("%s (%s:%d → %s) [%s] %s",
				tunnelName,
				tunnel.config.Host,
				tunnel.config.Port,
				tunnelAddr,
				tunnel.config.Subdomain,
				statusText,
			)
		}
		s += tunnelStyle.MaxWidth(lineWidth).Render(tunnelInfo) + "\n"
	}
	s += "\n"

	// The QR panel sits above the table because a short terminal clips the
	// bottom of the alt screen, and the table is the right thing to lose.
	if m.qrPanel != "" {
		s += m.qrPanel + "\n\n"
	}

	// Add waiting message if no logs
	if len(m.table.Rows()) == 0 {
		// Create empty table with just headers
		m.table.SetRows([]table.Row{{"", "", "", "", "Waiting for logs..."}})
	}
	s += tableStyle.Render(m.table.View()) + "\n"

	// Show debug table if debug mode is enabled
	if m.debug {
		s += "\n" + titleStyle.Render("🔍 Debug Logs") + "\n"
		if len(m.debugTable.Rows()) == 0 {
			// Create empty debug table with just headers
			m.debugTable.SetRows([]table.Row{{"", "", "Waiting for logs...", ""}})
		}
		s += tableStyle.Render(m.debugTable.View()) + "\n"
	}

	// Help and status
	help := "Ctrl+C: Quit"
	if m.qrEnabled {
		help += " • r: QR code"
	}
	s += "\n" + subtitleStyle.Render(help) + "\n"
	s += subtitleStyle.Render(fmt.Sprintf("Last updated: %s", m.lastUpdate.Format("15:04:05"))) + "\n"

	return s
}

func (m *model) AddLog(msg AddLogMsg) {
	// Clear waiting message if it exists
	if len(m.table.Rows()) == 1 && m.table.Rows()[0][4] == "Waiting for logs..." {
		m.table.SetRows([]table.Row{})
	}

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

	// Set styles only when we have actual log messages
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
	m.table.SetStyles(s)
}

// Add method to handle debug logs
func (m *model) AddDebugLog(msg AddDebugLogMsg) {
	if !m.debug {
		return
	}

	// Clear waiting message if it exists
	if len(m.debugTable.Rows()) == 1 && m.debugTable.Rows()[0][2] == "Waiting for logs..." {
		m.debugTable.SetRows([]table.Row{})
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
