package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Relay struct {
	Hostname   string `json:"hostname"`
	Location   string `json:"location"`
	Active     bool   `json:"active"`
	IPv4AddrIn string `json:"ipv4_addr_in"`
}

type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

type APIResponse struct {
	Locations map[string]Location `json:"locations"`
	Wireguard struct {
		Relays []Relay `json:"relays"`
	} `json:"wireguard"`
}

// Global style for the table
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	spinner spinner.Model
	loading bool
	table   table.Model
	err     error
}

// Implement the tea.Model interface
func (m model) Init() tea.Cmd {
	// Start the spinner and fetch data
	return tea.Batch(spinner.Tick, fetchAPIData)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case APIResponse:
		m.loading = false
		m.table = createTable(msg)
		return m, nil
	case error:
		m.loading = false
		m.err = msg
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	if !m.loading {
		m.table, _ = m.table.Update(msg)
	}

	return m, nil
}

func (m model) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s Fetching data...", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	return baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n"
}

func fetchAPIData() tea.Msg {

	url := "https://api.mullvad.net/app/v1/relays"

	// Fetch and parse JSON data
	apiResp, err := fetchAndParseJSON(url)
	if err != nil {
		return err
	}
	return apiResp
}

func fetchAndParseJSON(url string) (APIResponse, error) {
	// time.Sleep(2 * time.Second) // Simulate delay for demonstration
	resp, err := http.Get(url)
	if err != nil {
		return APIResponse{}, err
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return APIResponse{}, err
	}
	return apiResp, nil
}

func createTable(apiResp APIResponse) table.Model {
	// Define table columns
	columns := []table.Column{
		{Title: "Hostname", Width: 20},
		{Title: "Location", Width: 10},
		{Title: "Active", Width: 7},
		{Title: "IPv4 Address", Width: 15},
		{Title: "Country", Width: 15},
	}

	// Convert JSON data to table rows
	var rows []table.Row
	for _, relay := range apiResp.Wireguard.Relays {
		locationInfo := apiResp.Locations[relay.Location]
		rows = append(rows, table.Row{
			relay.Hostname,
			relay.Location,
			fmt.Sprintf("%t", relay.Active),
			relay.IPv4AddrIn,
			locationInfo.Country,
		})
	}

	// Create the table with the data
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(11),
	)

	// Define table styles
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

	return t
}

func main() {
	// Create the initial model with a spinner
	m := model{
		spinner: spinner.New(spinner.WithSpinner(spinner.Moon)),
		loading: true,
	}

	// Start the TUI program
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
