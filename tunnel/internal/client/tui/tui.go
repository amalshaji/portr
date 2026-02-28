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
	width      int
}

func New(debug bool) *tea.Program {
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
			tunnels:    make(map[string]*tunnelStatus),
			table:      t,
			debugTable: dt,
			debug:      debug,
			width:      80,
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
		}

	case ErrorMsg:
		m.err = msg.Error
		// Don't quit immediately, let the user see the error
		return m, nil

	case QuitMsg:
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
		m.width = msg.Width

		// Calculate dynamic widths based on terminal size
		totalWidth := max(msg.Width-6, 30) // Account for borders and terminal edges

		// Adjust table heights based on terminal height
		availableHeight := msg.Height - 15 // Account for headers and other UI elements
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
	lineWidth := max(m.width-4, 20)
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
		s += tunnelStyle.MaxWidth(lineWidth).Render(tunnelInfo) + "\n"
	}
	s += "\n"

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
	s += "\n" + subtitleStyle.Render("Ctrl+C: Quit") + "\n"
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
