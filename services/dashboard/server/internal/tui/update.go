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
			if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				m.cfgFocus = (m.cfgFocus + 1) % 3
			} else {
				m.activeTab = (m.activeTab + 1) % len(m.tabs)
			}

		case "shift+tab", "left":
			if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				m.cfgFocus = (m.cfgFocus - 1 + 3) % 3
			} else {
				m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			}

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
			} else if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				c := m.cfgCursors[m.cfgFocus]
				m.cfgCursors[m.cfgFocus] = max(0, c-1)
			}

		case "down", "j":
			if m.activeTab == 2 {
				m.selected = min(len(m.services)-1, m.selected+1)
			} else if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				n := m.cfgListLen()
				if n > 0 {
					c := m.cfgCursors[m.cfgFocus]
					m.cfgCursors[m.cfgFocus] = min(n-1, c+1)
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
				if m.cfgConfirmDel {
					m.cfgConfirmDel = false
					m.cfgRemoveItem()
				} else if m.cfgAdding {
					m.cfgAddItem()
				}
			} else {
				m.toggleSelectedService()
			}

		case "s":
			if !(m.activeTab == 4 && (m.cfgAdding || m.cfgConfirmDel)) {
				m.toggleSelectedService()
			}

		case "a":
			if m.activeTab == 4 && !m.cfgConfirmDel {
				m.cfgAdding = true
				m.cfgInput = ""
				m.cfgErr = ""
			} else if m.activeTab != 4 || !m.cfgConfirmDel {
				m.startAllServices()
			}

		case "x":
			if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				if m.cfgListLen() > 0 {
					m.cfgConfirmDel = true
					m.cfgErr = ""
				}
			} else if !(m.activeTab == 4 && (m.cfgAdding || m.cfgConfirmDel)) {
				m.stopAllServices()
			}

		case "r":
			if m.activeTab == 4 && !m.cfgAdding && !m.cfgConfirmDel {
				m.cfgReload()
			} else if !(m.activeTab == 4 && (m.cfgAdding || m.cfgConfirmDel)) {
				m.restartSelectedService()
			}

		case "y":
			if m.cfgConfirmDel {
				m.cfgConfirmDel = false
				m.cfgRemoveItem()
			}

		case "n":
			if m.cfgConfirmDel {
				m.cfgConfirmDel = false
				m.cfgErr = ""
			}

		case "esc":
			if m.cfgAdding {
				m.cfgAdding = false
				m.cfgInput = ""
				m.cfgErr = ""
			} else if m.cfgConfirmDel {
				m.cfgConfirmDel = false
				m.cfgErr = ""
			}

		case "ctrl+l":
			if m.activeTab == 1 {
				m.services[m.logViewSvc].Logs = NewRingBuffer(500)
			}

		case "backspace":
			if m.cfgAdding && len(m.cfgInput) > 0 {
				m.cfgInput = m.cfgInput[:len(m.cfgInput)-1]
			}

		default:
			if m.cfgAdding {
				if msg.Type == tea.KeyRunes {
					m.cfgInput += string(msg.Runes)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.frame = (m.frame + 1) % 40
		m.tickCount++
		if m.tickCount%5 == 0 {
			m.stats.shiftBuckets()
		}
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
		(strings.Contains(lower, "target") && strings.Contains(lower, "emitted")) ||
		strings.Contains(lower, "address watcher"):
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

// ── Config helpers ───────────────────────────────────────────────────

func (m *Model) cfgAddItem() {
	input := strings.TrimSpace(m.cfgInput)
	if input == "" {
		m.cfgErr = "cannot be empty"
		return
	}

	switch m.cfgFocus {
	case 0:
		parts := strings.Split(input, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			m.cfgErr = "invalid format (use owner/name)"
			return
		}
		if cfgContains(m.watchCfg.Repos, input) {
			m.cfgErr = "already watched"
			return
		}
		m.watchCfg.Repos = append(m.watchCfg.Repos, input)
		m.addEvent("Dashboard", "added repo "+input, "target")
	case 1:
		if !strings.HasPrefix(input, "0x") || len(input) != 42 {
			m.cfgErr = "invalid address (use 0x + 40 hex chars)"
			return
		}
		if cfgContains(m.watchCfg.Contracts, input) {
			m.cfgErr = "already watched"
			return
		}
		m.watchCfg.Contracts = append(m.watchCfg.Contracts, input)
		m.addEvent("Dashboard", "added contract "+input, "target")
	case 2:
		if !strings.HasPrefix(input, "0x") || len(input) != 42 {
			m.cfgErr = "invalid address (use 0x + 40 hex chars)"
			return
		}
		if cfgContains(m.watchCfg.Wallets, input) {
			m.cfgErr = "already watched"
			return
		}
		m.watchCfg.Wallets = append(m.watchCfg.Wallets, input)
		m.addEvent("Dashboard", "added wallet "+input, "target")
	}

	if m.watchCfgPath != "" {
		if err := saveRepoConfig(m.watchCfgPath, m.watchCfg); err != nil {
			m.cfgErr = "save failed: " + err.Error()
			return
		}
	}
	m.cfgAdding = false
	m.cfgInput = ""
	m.cfgErr = ""
}

func (m *Model) cfgRemoveItem() {
	c := m.cfgCursors[m.cfgFocus]
	var removed string
	switch m.cfgFocus {
	case 0:
		if c < 0 || c >= len(m.watchCfg.Repos) {
			return
		}
		removed = m.watchCfg.Repos[c]
		m.watchCfg.Repos = append(m.watchCfg.Repos[:c], m.watchCfg.Repos[c+1:]...)
	case 1:
		if c < 0 || c >= len(m.watchCfg.Contracts) {
			return
		}
		removed = m.watchCfg.Contracts[c]
		m.watchCfg.Contracts = append(m.watchCfg.Contracts[:c], m.watchCfg.Contracts[c+1:]...)
	case 2:
		if c < 0 || c >= len(m.watchCfg.Wallets) {
			return
		}
		removed = m.watchCfg.Wallets[c]
		m.watchCfg.Wallets = append(m.watchCfg.Wallets[:c], m.watchCfg.Wallets[c+1:]...)
	}

	if m.watchCfgPath != "" {
		if err := saveRepoConfig(m.watchCfgPath, m.watchCfg); err != nil {
			m.cfgErr = "save failed: " + err.Error()
			return
		}
	}

	if m.cfgCursors[m.cfgFocus] >= m.cfgListLen() && m.cfgListLen() > 0 {
		m.cfgCursors[m.cfgFocus] = m.cfgListLen() - 1
	}
	m.addEvent("Dashboard", "removed "+removed, "target")
}

func (m *Model) cfgReload() {
	if m.watchCfgPath == "" {
		m.cfgErr = "no config path known"
		return
	}
	cfg, err := loadRepoConfig(m.watchCfgPath)
	if err != nil {
		m.cfgErr = "reload failed: " + err.Error()
		return
	}
	m.watchCfg = cfg
	m.cfgCursors = [3]int{0, 0, 0}
	m.cfgErr = ""
	m.addEvent("Dashboard", "reloaded watch config", "target")
}

func cfgContains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
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
