package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginLeft(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("246"))

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("63"))

	logPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(1)

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))
)

func statusBadge(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render("● RUNNING")
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("○ STOPPED")
	case "booting":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("◐ BOOTING")
	case "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("✗ ERROR")
	}
	return status
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("🐝 SWARM DASHBOARD"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Autonomous Web3 Security Swarm — Terminal UI"))
	b.WriteString("\n\n")

	// Tabs
	var tabs []string
	for i, t := range m.tabs {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(t))
		} else {
			tabs = append(tabs, tabStyle.Render(t))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	b.WriteString("\n\n")

	// Content area
	contentHeight := m.height - 10
	if contentHeight < 5 {
		contentHeight = 5
	}

	switch m.activeTab {
	case 0:
		b.WriteString(m.renderOverview(contentHeight))
	case 1:
		b.WriteString(m.renderLogs(contentHeight))
	case 2:
		b.WriteString(m.renderServices(contentHeight))
	}

	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderOverview(h int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).MarginLeft(1).Render("Pipeline Overview") + "\n\n")

	counts := map[string]int{"running": 0, "stopped": 0, "booting": 0, "error": 0}
	for _, svc := range m.services {
		counts[svc.Status]++
	}

	stats := []string{
		fmt.Sprintf("  Total Services: %d", len(m.services)),
		fmt.Sprintf("  Running:    %s", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(fmt.Sprintf("%d", counts["running"]))),
		fmt.Sprintf("  Stopped:    %s", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("%d", counts["stopped"]))),
		fmt.Sprintf("  Booting:    %s", lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(fmt.Sprintf("%d", counts["booting"]))),
		fmt.Sprintf("  Error:      %s", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf("%d", counts["error"]))),
	}
	b.WriteString(strings.Join(stats, "\n"))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).MarginLeft(1).Render("Active Agents") + "\n\n")
	for _, svc := range m.services {
		b.WriteString(fmt.Sprintf("  %-14s %s\n", svc.Name, statusBadge(svc.Status)))
	}

	return logPaneStyle.Render(b.String())
}

func (m Model) renderLogs(h int) string {
	svc := m.services[m.selected]
	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Logs: %s", svc.Name))

	logText := svc.Logs.String()
	lines := strings.Split(logText, "\n")

	// Keep only last lines that fit
	maxLines := h - 4
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	content := strings.Join(lines, "\n")
	if content == "" {
		content = "  (no logs yet)"
	}

	return fmt.Sprintf("%s\n%s", header, logPaneStyle.Height(h).Render(content))
}

func (m Model) renderServices(h int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).MarginLeft(1).Render("Services") + "\n\n")

	for i, svc := range m.services {
		prefix := "  "
		if i == m.selected {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true).Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%-14s %s  %s\n",
			prefix,
			svc.Name,
			statusBadge(svc.Status),
			lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(strings.Join(svc.Cmd, " ")),
		))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"  Use ↑/↓ or j/k to select, Enter/s to toggle, r to restart"))

	return logPaneStyle.Render(b.String())
}

func (m Model) renderFooter() string {
	hints := []string{
		fmt.Sprintf("%s tab ⇄ %s", keyStyle.Render("1-3"), keyStyle.Render("←→")),
		fmt.Sprintf("%s start/stop", keyStyle.Render("s/Enter")),
		fmt.Sprintf("%s start all", keyStyle.Render("a")),
		fmt.Sprintf("%s stop all", keyStyle.Render("x")),
		fmt.Sprintf("%s restart", keyStyle.Render("r")),
		fmt.Sprintf("%s quit", keyStyle.Render("q")),
	}
	return hintStyle.Render("  " + strings.Join(hints, "  •  "))
}
