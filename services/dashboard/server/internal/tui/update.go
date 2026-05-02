package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.shutdownAll()
			return m, tea.Quit

		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)

		case "shift+tab", "left":
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)

		case "1":
			m.activeTab = 0
		case "2":
			m.activeTab = 1
		case "3":
			m.activeTab = 2
		case "4":
			m.activeTab = 3

		case "up", "k":
			if m.activeTab == 2 {
				m.selected = max(0, m.selected-1)
			}

		case "down", "j":
			if m.activeTab == 2 {
				m.selected = min(len(m.services)-1, m.selected+1)
			}

		case "[", "h":
			if m.activeTab == 1 {
				m.logViewSvc = (m.logViewSvc - 1 + len(m.services)) % len(m.services)
			}

		case "]", "l", "/":
			if m.activeTab == 1 {
				m.logViewSvc = (m.logViewSvc + 1) % len(m.services)
			}

		case "enter", "s":
			m.toggleSelectedService()

		case "a":
			m.startAllServices()

		case "x":
			m.stopAllServices()

		case "r":
			m.restartSelectedService()

		case "ctrl+l":
			if m.activeTab == 1 {
				m.services[m.logViewSvc].Logs = NewRingBuffer(500)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Advance animation frame (0-39)
		m.frame = (m.frame + 1) % 40
		m.tickCount++

		// Every 5 ticks (~1 second) shift sparkline buckets
		if m.tickCount%5 == 0 {
			m.stats.shiftBuckets()
		}

		// Promote booting → running after 3 seconds
		for _, svc := range m.services {
			if svc.Status == "booting" && svc.Process != nil && time.Since(svc.Process.started) > 3*time.Second {
				svc.Status = "running"
			}
		}
		return m, tickCmd()

	case processLogMsg:
		msg.Service.Logs.Append(msg.Line)
		msg.Service.LogCount++
		m.parseLogLine(msg.Service.Name, msg.Line)
		return m, m.listenCmd()

	case processExitMsg:
		if msg.Err != nil {
			msg.Service.Status = "error"
			msg.Service.Logs.Append("EXIT ERROR: " + msg.Err.Error())
		} else {
			msg.Service.Status = "stopped"
		}
		return m, m.listenCmd()

	default:
		return m, m.listenCmd()
	}

	return m, nil
}

// parseLogLine inspects a log line and updates pipeline stats / events accordingly.
func (m *Model) parseLogLine(svcName, line string) {
	lower := strings.ToLower(line)

	switch {
	case strings.Contains(lower, "exploit generated"):
		m.stats.Exploits++
		m.stats.exploitAcc++
		m.addEvent(svcName, line, "exploit")

	case strings.Contains(lower, "disclosure published"):
		m.stats.Disclosures++
		m.stats.disclosureAcc++
		m.addEvent(svcName, line, "disclosure")

	case strings.Contains(lower, "findings for target") || strings.Contains(lower, "publish_finding"):
		m.stats.Findings++
		m.stats.findingAcc++
		m.addEvent(svcName, line, "finding")

	case strings.Contains(lower, "published target") ||
		(strings.Contains(lower, "target") && strings.Contains(lower, "discovered")):
		m.stats.Targets++
		m.stats.targetAcc++
		m.addEvent(svcName, line, "target")

	case strings.Contains(lower, "error") && !strings.Contains(lower, "no error"):
		m.addEvent(svcName, line, "error")

	case strings.Contains(lower, "warn"):
		m.addEvent(svcName, line, "warn")
	}
}

// ── Service control helpers ──────────────────────────────────────────

func (m *Model) toggleSelectedService() {
	svc := m.services[m.selected]
	switch svc.Status {
	case "running", "booting":
		m.stopService(svc)
	case "stopped", "error":
		m.startService(svc)
	}
}

func (m *Model) restartSelectedService() {
	svc := m.services[m.selected]
	m.stopService(svc)
	time.Sleep(200 * time.Millisecond)
	m.startService(svc)
}

func (m *Model) startAllServices() {
	for _, svc := range m.services {
		if svc.Status == "stopped" || svc.Status == "error" {
			m.startService(svc)
		}
	}
}

func (m *Model) stopAllServices() {
	for _, svc := range m.services {
		m.stopService(svc)
	}
}

func (m *Model) shutdownAll() {
	m.stopAllServices()
	time.Sleep(300 * time.Millisecond)
}

func (m *Model) startService(svc *Service) {
	if svc.Status == "running" || svc.Status == "booting" {
		return
	}
	svc.Status = "booting"
	svc.StartedAt = time.Now()
	go startProcess(svc, m.msgCh)
}

func (m *Model) stopService(svc *Service) {
	if svc.Process != nil {
		svc.Process.Stop()
		svc.Process = nil
	}
	svc.Status = "stopped"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
