package tui

import (
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
		case "up", "k":
			if m.activeTab == 2 {
				m.selected = max(0, m.selected-1)
			}
		case "down", "j":
			if m.activeTab == 2 {
				m.selected = min(len(m.services)-1, m.selected+1)
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
				m.services[m.selected].Logs = NewRingBuffer(500)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		for _, svc := range m.services {
			if svc.Status == "booting" && svc.Process != nil && time.Since(svc.Process.started) > 3*time.Second {
				svc.Status = "running"
			}
		}
		return m, tickCmd()

	case processLogMsg:
		msg.Service.Logs.Append(msg.Line)
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

// Service control helpers ────────────────────────────────────────────

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
	go startProcess(svc, m.msgCh)
}

func (m *Model) stopService(svc *Service) {
	if svc.Process != nil {
		svc.Process.Stop()
		svc.Process = nil
	}
	svc.Status = "stopped"
}

func max(a, b int) int { if a > b { return a }; return b }
func min(a, b int) int { if a < b { return a }; return b }
