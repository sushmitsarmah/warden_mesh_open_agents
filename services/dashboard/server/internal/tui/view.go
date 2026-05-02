package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── Color palette ────────────────────────────────────────────────────

var (
	colorAccent  = lipgloss.Color("86")  // cyan-green (header)
	colorGreen   = lipgloss.Color("42")  // running
	colorOrange  = lipgloss.Color("208") // booting
	colorRed     = lipgloss.Color("196") // error
	colorGray    = lipgloss.Color("241") // dim
	colorBlue    = lipgloss.Color("63")  // active tab bg
	colorYellow  = lipgloss.Color("220") // findings
	colorCyan    = lipgloss.Color("87")  // disclosures / exploits
	colorPurple  = lipgloss.Color("135") // orchestrator accent
)

// per-service accent colors (indexed by Service.Index)
var serviceColors = []lipgloss.Color{
	colorAccent,  // 0 AXL Node
	colorGreen,   // 1 Scout
	colorYellow,  // 2 Auditor
	colorPurple,  // 3 Orchestrator
}

// ── Styles ───────────────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("246"))

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(colorBlue)

	logPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBlue).
			Padding(0, 1)

	hintStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

	dimStyle = lipgloss.NewStyle().Foreground(colorGray)
)

// ── Spinner / animation constants ───────────────────────────────────

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// arrowChars is the arrow template. Width = 7.
const arrowTemplate = " ──►── "

// ── Helper functions ─────────────────────────────────────────────────

// clamp returns v clamped to [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// fmtDuration formats a duration as "MM:SS" or "H:MM:SS".
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// sparkline renders a [20]int bucket array as a bar using block chars.
func sparkline(buckets [20]int, width int) string {
	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find max value for scaling
	maxVal := 1
	for _, v := range buckets {
		if v > maxVal {
			maxVal = v
		}
	}

	// Use only the last `width` buckets
	start := 0
	if width < 20 {
		start = 20 - width
	}
	usable := buckets[start:]

	var sb strings.Builder
	for _, v := range usable {
		idx := int(float64(v) / float64(maxVal) * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

// ── Animated status badge ─────────────────────────────────────────────

func (m Model) animatedStatus(svc *Service) string {
	switch svc.Status {
	case "running":
		// Pulse between bright and dim green every 4 frames
		if (m.frame/4)%2 == 0 {
			return lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("● RUNNING")
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Render("● RUNNING")

	case "booting":
		sp := spinnerFrames[(m.frame/2)%10]
		return lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(sp + " BOOTING")

	case "error":
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("✗ ERROR")

	case "stopped":
		return dimStyle.Render("○ STOPPED")
	}
	return svc.Status
}

// ── Main View ─────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// ── Header ──────────────────────────────────────────────────────
	title := headerStyle.Render("WARDEN MESH")
	subtitle := subtitleStyle.Render("Autonomous Web3 Security Swarm")
	now := dimStyle.Render(time.Now().Format("15:04:05"))

	if m.width > 0 {
		// right-align the clock
		titlePart := lipgloss.JoinHorizontal(lipgloss.Top, " ", title, "  ", subtitle)
		headerLine := lipgloss.Place(m.width, 1, lipgloss.Left, lipgloss.Top,
			titlePart,
			lipgloss.WithWhitespaceChars(" "),
		)
		// overlay clock on right — build manually
		titleWidth := lipgloss.Width(titlePart)
		clockWidth := lipgloss.Width(now)
		pad := m.width - titleWidth - clockWidth
		if pad > 0 {
			headerLine = titlePart + strings.Repeat(" ", pad) + now
		} else {
			headerLine = titlePart
		}
		b.WriteString(headerLine)
	} else {
		b.WriteString(" " + title + "  " + subtitle)
	}
	b.WriteString("\n\n")

	// ── Tabs ─────────────────────────────────────────────────────────
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

	// ── Content ───────────────────────────────────────────────────────
	// header=1, blank=1, tabs=1, blank=1, footer=1, blank before footer=1 = 6
	contentHeight := m.height - 6
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

// ── Tab 1: Overview ───────────────────────────────────────────────────

func (m Model) renderOverview(h int) string {
	var sections []string

	sections = append(sections, m.renderPipeline())
	sections = append(sections, m.renderMetricCards())

	// Remaining height for events
	usedLines := 0
	for _, s := range sections {
		usedLines += strings.Count(s, "\n") + 1
	}
	eventsHeight := h - usedLines - 2
	if eventsHeight < 3 {
		eventsHeight = 3
	}
	sections = append(sections, m.renderEventsFeed(eventsHeight))

	return strings.Join(sections, "\n")
}

// renderPipeline renders the 4-box pipeline diagram with animated arrows.
func (m Model) renderPipeline() string {
	w := m.width
	if w == 0 {
		w = 80
	}

	// box width: (total - padding - 3 arrows * 7) / 4, clamped 13..20
	arrowWidth := 7
	boxW := (w - 4 - 3*arrowWidth) / 4
	boxW = clamp(boxW, 13, 20)
	innerW := boxW - 2 // subtract border chars

	var parts []string

	for i, svc := range m.services {
		// Determine border color
		var borderColor lipgloss.Color
		switch svc.Status {
		case "running":
			borderColor = colorGreen
		case "booting":
			borderColor = colorOrange
		case "error":
			borderColor = colorRed
		default:
			borderColor = colorGray
		}

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(innerW).
			Padding(0, 1)

		// Service name (bold, truncated)
		name := svc.Name
		if len(name) > innerW-2 {
			name = name[:innerW-2]
		}
		nameStr := lipgloss.NewStyle().Bold(true).Foreground(serviceColors[i]).Render(name)

		// Animated status
		statusStr := m.animatedStatus(svc)

		// Uptime line
		uptimeLine := ""
		if svc.Status == "running" || svc.Status == "booting" {
			if !svc.StartedAt.IsZero() {
				uptimeLine = dimStyle.Render("up " + fmtDuration(time.Since(svc.StartedAt)))
			}
		}

		// Per-service stat
		statLine := ""
		switch i {
		case 1: // Scout -> targets
			statLine = dimStyle.Render(fmt.Sprintf("tgt:%d", m.stats.Targets))
		case 2: // Auditor -> findings
			statLine = dimStyle.Render(fmt.Sprintf("fnd:%d", m.stats.Findings))
		case 3: // Orchestrator -> exploits
			statLine = dimStyle.Render(fmt.Sprintf("exp:%d", m.stats.Exploits))
		}

		// Build box content lines
		var lines []string
		lines = append(lines, nameStr)
		lines = append(lines, statusStr)
		if uptimeLine != "" {
			lines = append(lines, uptimeLine)
		}
		if statLine != "" {
			lines = append(lines, statLine)
		}

		boxContent := strings.Join(lines, "\n")
		parts = append(parts, boxStyle.Render(boxContent))

		// Add animated arrow between boxes (not after last)
		if i < len(m.services)-1 {
			parts = append(parts, m.renderArrow(svc, m.services[i+1], arrowWidth))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// renderArrow renders the animated arrow between two service boxes.
func (m Model) renderArrow(left, right *Service, width int) string {
	chars := []rune(arrowTemplate)
	arrowLen := len(chars)

	bothRunning := left.Status == "running" && right.Status == "running"

	if bothRunning {
		// Animate a bright accent char moving left-to-right
		pos := (m.frame / 2) % arrowLen
		var sb strings.Builder
		for i, ch := range chars {
			if i == pos {
				sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(string(ch)))
			} else {
				sb.WriteString(dimStyle.Render(string(ch)))
			}
		}
		return sb.String()
	}

	// Dim static arrow
	return dimStyle.Render(arrowTemplate)
}

// renderMetricCards renders 4 metric cards side-by-side.
func (m Model) renderMetricCards() string {
	w := m.width
	if w == 0 {
		w = 80
	}

	// card width: (total - 4 - 3*2 gap) / 4, clamped 14..22
	cardW := (w - 4 - 3*2) / 4
	cardW = clamp(cardW, 14, 22)
	innerW := cardW - 2

	type cardDef struct {
		label   string
		value   int
		color   lipgloss.Color
		buckets [20]int
	}

	cards := []cardDef{
		{"TARGETS", m.stats.Targets, colorGreen, m.stats.TargetBuckets},
		{"FINDINGS", m.stats.Findings, colorYellow, m.stats.FindingBuckets},
		{"EXPLOITS", m.stats.Exploits, colorRed, m.stats.ExploitBuckets},
		{"DISCLOSURES", m.stats.Disclosures, colorCyan, m.stats.DisclosureBuckets},
	}

	var cardStrs []string
	for i, c := range cards {
		cardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.color).
			Width(innerW).
			Padding(0, 1)

		label := dimStyle.Render(c.label)
		number := lipgloss.NewStyle().Bold(true).Foreground(c.color).Render(fmt.Sprintf("%d", c.value))
		spark := lipgloss.NewStyle().Foreground(c.color).Render(sparkline(c.buckets, innerW-2))

		content := label + "\n" + number + "\n" + spark
		cardStrs = append(cardStrs, cardStyle.Render(content))

		// 2-space gap between cards (except last)
		if i < len(cards)-1 {
			cardStrs = append(cardStrs, "  ")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cardStrs...)
}

// renderEventsFeed renders the recent events at the bottom of the overview.
func (m Model) renderEventsFeed(height int) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render("RECENT EVENTS")

	// How many events fit
	maxEvents := height - 2
	if maxEvents < 1 {
		maxEvents = 1
	}

	evs := m.events
	if len(evs) > maxEvents {
		evs = evs[len(evs)-maxEvents:]
	}

	var lines []string
	lines = append(lines, header)

	w := m.width
	if w == 0 {
		w = 80
	}
	// approximate usable width for text after timestamp+svc+badge
	textWidth := w - 10 - 8 - 6 - 4
	if textWidth < 10 {
		textWidth = 10
	}

	for _, ev := range evs {
		ts := dimStyle.Render(ev.Time)
		svcName := fmt.Sprintf("%-6s", truncate(ev.Service, 6))
		svcIdx := svcIndexByName(m.services, ev.Service)
		if svcIdx >= 0 && svcIdx < len(serviceColors) {
			svcName = lipgloss.NewStyle().Foreground(serviceColors[svcIdx]).Render(svcName)
		} else {
			svcName = dimStyle.Render(svcName)
		}

		badge := kindBadge(ev.Kind)
		text := truncate(ev.Text, textWidth)

		lines = append(lines, ts+" "+svcName+" "+badge+" "+text)
	}

	if len(evs) == 0 {
		lines = append(lines, dimStyle.Render("  (no events yet)"))
	}

	return strings.Join(lines, "\n")
}

// kindBadge returns a colored badge string for an event kind.
func kindBadge(kind string) string {
	switch kind {
	case "target":
		return lipgloss.NewStyle().Foreground(colorGreen).Render("[TGT]")
	case "finding":
		return lipgloss.NewStyle().Foreground(colorYellow).Render("[FND]")
	case "exploit":
		return lipgloss.NewStyle().Foreground(colorRed).Render("[EXP]")
	case "disclosure":
		return lipgloss.NewStyle().Foreground(colorCyan).Render("[DSC]")
	case "error":
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("[ERR]")
	case "warn":
		return lipgloss.NewStyle().Foreground(colorOrange).Render("[WRN]")
	}
	return dimStyle.Render("[   ]")
}

// svcIndexByName returns the Index of the service with given name, or -1.
func svcIndexByName(services []*Service, name string) int {
	for _, s := range services {
		if s.Name == name {
			return s.Index
		}
	}
	return -1
}

// truncate truncates s to at most n runes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// ── Tab 2: Logs ───────────────────────────────────────────────────────

func (m Model) renderLogs(h int) string {
	// Mini-tab row
	var miniTabs []string
	for i, svc := range m.services {
		if i == m.logViewSvc {
			miniTabs = append(miniTabs, activeTabStyle.Render(svc.Name))
		} else {
			miniTabs = append(miniTabs, tabStyle.Render(svc.Name))
		}
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, miniTabs...)
	hint := dimStyle.Render("  / [ ] to switch service")

	svc := m.services[m.logViewSvc]
	allLines := svc.Logs.Lines()

	// Reserve lines: tabRow=1, hint=1, border top/bottom=2, padding=0
	maxLines := h - 4
	if maxLines < 1 {
		maxLines = 1
	}
	if len(allLines) > maxLines {
		allLines = allLines[len(allLines)-maxLines:]
	}

	var colorized []string
	for _, l := range allLines {
		colorized = append(colorized, colorizeLogLine(l))
	}

	content := strings.Join(colorized, "\n")
	if content == "" {
		content = dimStyle.Render("  (no logs yet)")
	}

	pane := logPaneStyle.
		Width(m.width - 4).
		Height(maxLines).
		Render(content)

	return tabRow + "\n" + hint + "\n" + pane
}

// colorizeLogLine applies color to a log line based on content.
func colorizeLogLine(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "fatal"):
		return lipgloss.NewStyle().Foreground(colorRed).Render(line)
	case strings.Contains(lower, "warn"):
		return lipgloss.NewStyle().Foreground(colorYellow).Render(line)
	case strings.HasPrefix(lower, "15:") || strings.Contains(line, "[stderr]"):
		// Check for [stderr] prefix after possible timestamp
		if strings.Contains(line, "[stderr]") {
			return lipgloss.NewStyle().Foreground(colorOrange).Render(line)
		}
		fallthrough
	case strings.Contains(lower, "exploit generated") || strings.Contains(lower, "disclosure published"):
		return lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(line)
	case strings.Contains(lower, "finding") || strings.Contains(lower, "target"):
		return lipgloss.NewStyle().Foreground(colorGreen).Render(line)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(line)
	}
}

// ── Tab 3: Services ───────────────────────────────────────────────────

func (m Model) renderServices(h int) string {
	var b strings.Builder

	for i, svc := range m.services {
		var prefix string
		if i == m.selected {
			prefix = lipgloss.NewStyle().Foreground(colorBlue).Bold(true).Render("▶")
		} else {
			prefix = " "
		}

		name := fmt.Sprintf("%-14s", truncate(svc.Name, 14))
		if i == m.selected {
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}

		status := m.animatedStatus(svc)

		uptime := ""
		if (svc.Status == "running" || svc.Status == "booting") && !svc.StartedAt.IsZero() {
			uptime = dimStyle.Render(" up:" + fmtDuration(time.Since(svc.StartedAt)))
		}

		logCount := dimStyle.Render(fmt.Sprintf(" logs:%d", svc.LogCount))

		b.WriteString(fmt.Sprintf(" %s %s  %s%s%s\n", prefix, name, status, uptime, logCount))
	}

	// Selected service details
	b.WriteString("\n")
	sel := m.services[m.selected]
	b.WriteString(dimStyle.Render("  cmd: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(strings.Join(sel.Cmd, " ")) + "\n")
	b.WriteString(dimStyle.Render("  dir: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(sel.Dir) + "\n")

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  ↑/↓ j/k select  •  Enter/s toggle  •  r restart  •  a start all  •  x stop all"))

	return logPaneStyle.Render(b.String())
}

// ── Footer ────────────────────────────────────────────────────────────

func (m Model) renderFooter() string {
	hints := []string{
		keyStyle.Render("1-3") + dimStyle.Render(" tabs"),
		keyStyle.Render("←→/Tab") + dimStyle.Render(" switch"),
		keyStyle.Render("s") + dimStyle.Render(" start/stop"),
		keyStyle.Render("a") + dimStyle.Render(" start all"),
		keyStyle.Render("x") + dimStyle.Render(" stop all"),
		keyStyle.Render("r") + dimStyle.Render(" restart"),
		keyStyle.Render("[/]") + dimStyle.Render(" log svc"),
		keyStyle.Render("q") + dimStyle.Render(" quit"),
	}
	return hintStyle.Render("  " + strings.Join(hints, "  "))
}
