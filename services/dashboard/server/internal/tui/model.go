package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Messages ────────────────────────────────────────────────────────

type tickMsg time.Time

type processLogMsg struct {
	Service *Service
	Line    string
}

type processExitMsg struct {
	Service *Service
	Err     error
}

// ── RingBuffer ──────────────────────────────────────────────────────

type RingBuffer struct {
	mu     sync.Mutex
	lines  []string
	size   int
	cursor int
	full   bool
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{lines: make([]string, size), size: size}
}

func (r *RingBuffer) Append(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ts := time.Now().Format("15:04:05")
	r.lines[r.cursor] = fmt.Sprintf("%s %s", ts, line)
	r.cursor = (r.cursor + 1) % r.size
	if r.cursor == 0 {
		r.full = true
	}
}

func (r *RingBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return strings.Join(r.lines[:r.cursor], "\n")
	}
	var out []string
	for i := 0; i < r.size; i++ {
		idx := (r.cursor + i) % r.size
		if r.lines[idx] != "" {
			out = append(out, r.lines[idx])
		}
	}
	return strings.Join(out, "\n")
}

// ── Service ─────────────────────────────────────────────────────────

type Service struct {
	Name    string
	Cmd     []string
	Dir     string
	Status  string // running, stopped, booting, error
	Logs    *RingBuffer
	Process *Process
	Index   int
}

func defaultServices() []*Service {
	return []*Service{
		{Index: 0, Name: "AXL Node", Cmd: []string{"./node", "-config", "node-config.json"}, Dir: "../../../axl", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 1, Name: "Scout", Cmd: []string{"go", "run", "./cmd"}, Dir: "../../scout-go", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 2, Name: "Auditor", Cmd: []string{"cargo", "run", "--release", "--bin", "auditor"}, Dir: "../../auditor-rs", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 3, Name: "Orchestrator", Cmd: []string{"go", "run", "./cmd"}, Dir: "../../orchestrator-go", Status: "stopped", Logs: NewRingBuffer(500)},
	}
}

// ── Model ───────────────────────────────────────────────────────────

type Model struct {
	tabs       []string
	activeTab  int
	services   []*Service
	selected   int // selected service index on Services tab
	width      int
	height     int
	msgCh      chan tea.Msg
	quitting   bool
}

func NewModel() Model {
	return Model{
		tabs:     []string{"Overview", "Logs", "Services"},
		services: defaultServices(),
		selected: 0,
		msgCh:    make(chan tea.Msg, 256),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.listenCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) listenCmd() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgCh
	}
}
