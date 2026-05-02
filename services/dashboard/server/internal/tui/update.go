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
		case "5":
			m.activeTab = 4

		case "up", "k":
			if m.activeTab == 2 {
				m.selected = max(0, m.selected-1)
			} else if m.activeTab == 4 && !m.addingRepo && !m.confirmDelete {
				m.repoCursor = max(0, m.repoCursor-1)
			}

		case "down", "j":
			if m.activeTab == 2 {
				m.selected = min(len(m.services)-1, m.selected+1)
			} else if m.activeTab == 4 && !m.addingRepo && !m.confirmDelete {
				repoCount := len(m.repoCfg.Repos)
				if repoCount == 0 {
					m.repoCursor = 0
				} else {
					m.repoCursor = min(repoCount-1, m.repoCursor+1)
				}
			}

		case "[", "h":
			if m.activeTab == 1 {
				m.logViewSvc = (m.logViewSvc - 1 + len(m.services)) % len(m.services)
			}

		case "]", "l", "/":
			if m.activeTab == 1 {
				m.logViewSvc = (m.logViewSvc + 1) % len(m.services)
			}

		case "enter":
			if m.activeTab == 4 {
				if m.confirmDelete {
					m.confirmDelete = false
					m.removeSelectedRepo()
				} else if m.addingRepo {
					m.addNewRepo()
				}
			} else {
				m.toggleSelectedService()
			}

		case "s":
			if !(m.activeTab == 4 && (m.addingRepo || m.confirmDelete)) {
				m.toggleSelectedService()
			}

		case "a":
			if m.activeTab == 4 && !m.confirmDelete {
				m.addingRepo = true
				m.repoInput = ""
				m.repoErr = ""
			} else if m.activeTab != 4 || !m.confirmDelete {
				m.startAllServices()
			}

		case "x":
			if m.activeTab == 4 && !m.addingRepo && !m.confirmDelete {
				if len(m.repoCfg.Repos) > 0 {
					m.confirmDelete = true
					m.repoErr = ""
				}
			} else if !(m.activeTab == 4 && (m.addingRepo || m.confirmDelete)) {
				m.stopAllServices()
			}

		case "r":
			if m.activeTab == 4 && !m.addingRepo && !m.confirmDelete {
				m.reloadRepoConfig()
			} else if !(m.activeTab == 4 && (m.addingRepo || m.confirmDelete)) {
				m.restartSelectedService()
			}

		case "y":
			if m.confirmDelete {
				m.confirmDelete = false
				m.removeSelectedRepo()
			}

		case "n":
			if m.confirmDelete {
				m.confirmDelete = false
				m.repoErr = ""
			}

		case "esc":
			if m.addingRepo {
				m.addingRepo = false
				m.repoInput = ""
				m.repoErr = ""
			} else if m.confirmDelete {
				m.confirmDelete = false
				m.repoErr = ""
			}

		case "ctrl+l":
			if m.activeTab == 1 {
				m.services[m.logViewSvc].Logs = NewRingBuffer(500)
			}

		case "backspace":
			if m.addingRepo {
				if len(m.repoInput) > 0 {
					m.repoInput = m.repoInput[:len(m.repoInput)-1]
				}
			}

		default:
			if m.addingRepo {
				if msg.Type == tea.KeyRunes {
					m.repoInput += string(msg.Runes)
				}
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

// ── Repo config helpers ──────────────────────────────────────────────

func (m *Model) addNewRepo() {
	input := strings.TrimSpace(m.repoInput)
	if input == "" {
		m.repoErr = "repo cannot be empty"
		return
	}
	// Basic validation: must be owner/name format
	parts := strings.Split(input, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		m.repoErr = "invalid format (use owner/name)"
		return
	}
	// Dedupe
	for _, r := range m.repoCfg.Repos {
		if r == input {
			m.repoErr = "repo already watched"
			return
		}
	}
	m.repoCfg.Repos = append(m.repoCfg.Repos, input)
	if m.repoCfgPath != "" {
		if err := saveRepoConfig(m.repoCfgPath, m.repoCfg); err != nil {
			m.repoErr = "save failed: " + err.Error()
			return
		}
	}
	m.addingRepo = false
	m.repoInput = ""
	m.repoErr = ""
	m.addEvent("Dashboard", "added repo "+input, "target")
}

func (m *Model) removeSelectedRepo() {
	if m.repoCursor < 0 || m.repoCursor >= len(m.repoCfg.Repos) {
		return
	}
	removed := m.repoCfg.Repos[m.repoCursor]
	m.repoCfg.Repos = append(m.repoCfg.Repos[:m.repoCursor], m.repoCfg.Repos[m.repoCursor+1:]...)
	if m.repoCfgPath != "" {
		if err := saveRepoConfig(m.repoCfgPath, m.repoCfg); err != nil {
			m.repoErr = "save failed: " + err.Error()
			return
		}
	}
	if m.repoCursor >= len(m.repoCfg.Repos) && len(m.repoCfg.Repos) > 0 {
		m.repoCursor = len(m.repoCfg.Repos) - 1
	}
	m.addEvent("Dashboard", "removed repo "+removed, "target")
}

func (m *Model) reloadRepoConfig() {
	if m.repoCfgPath == "" {
		m.repoErr = "no config path known"
		return
	}
	cfg, err := loadRepoConfig(m.repoCfgPath)
	if err != nil {
		m.repoErr = "reload failed: " + err.Error()
		return
	}
	m.repoCfg = cfg
	m.repoCursor = 0
	m.repoErr = ""
	m.addEvent("Dashboard", "reloaded repo config", "target")
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
