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

// Lines returns all non-empty lines in order (oldest first).
func (r *RingBuffer) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	if !r.full {
		for _, l := range r.lines[:r.cursor] {
			if l != "" {
				out = append(out, l)
			}
		}
		return out
	}
	for i := 0; i < r.size; i++ {
		idx := (r.cursor + i) % r.size
		if r.lines[idx] != "" {
			out = append(out, r.lines[idx])
		}
	}
	return out
}

// ── PipelineStats ───────────────────────────────────────────────────

type PipelineStats struct {
	Targets     int
	Findings    int
	Exploits    int
	Disclosures int

	TargetBuckets     [20]int
	FindingBuckets    [20]int
	ExploitBuckets    [20]int
	DisclosureBuckets [20]int

	targetAcc     int
	findingAcc    int
	exploitAcc    int
	disclosureAcc int
}

func (s *PipelineStats) shiftBuckets() {
	for i := 0; i < 19; i++ {
		s.TargetBuckets[i] = s.TargetBuckets[i+1]
		s.FindingBuckets[i] = s.FindingBuckets[i+1]
		s.ExploitBuckets[i] = s.ExploitBuckets[i+1]
		s.DisclosureBuckets[i] = s.DisclosureBuckets[i+1]
	}
	s.TargetBuckets[19] = s.targetAcc
	s.FindingBuckets[19] = s.findingAcc
	s.ExploitBuckets[19] = s.exploitAcc
	s.DisclosureBuckets[19] = s.disclosureAcc
	s.targetAcc = 0
	s.findingAcc = 0
	s.exploitAcc = 0
	s.disclosureAcc = 0
}

// ── EventLine ───────────────────────────────────────────────────────

type EventLine struct {
	Time    string
	Service string
	Text    string
	Kind    string
}

// ── Service ─────────────────────────────────────────────────────────

type Service struct {
	Name      string
	Cmd       []string
	Dir       string
	Status    string
	Logs      *RingBuffer
	Process   *Process
	Index     int
	StartedAt time.Time
	LogCount  int
}

func defaultServices() []*Service {
	return []*Service{
		{Index: 0, Name: "AXL Node A", Cmd: []string{"./node", "-config", "node-config-a.json"}, Dir: "../../../axl", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 1, Name: "AXL Node B", Cmd: []string{"./node", "-config", "node-config-b.json"}, Dir: "../../../axl", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 2, Name: "Scout", Cmd: []string{"go", "run", "./cmd"}, Dir: "../../scout-go", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 3, Name: "Auditor", Cmd: []string{"cargo", "run", "--release", "--bin", "auditor"}, Dir: "../../auditor-rs", Status: "stopped", Logs: NewRingBuffer(500)},
		{Index: 4, Name: "Orchestrator", Cmd: []string{"go", "run", "./cmd"}, Dir: "../../orchestrator-go", Status: "stopped", Logs: NewRingBuffer(500)},
	}
}

// ── Model ───────────────────────────────────────────────────────────

type Model struct {
	tabs        []string
	activeTab   int
	services    []*Service
	selected    int
	logViewSvc  int
	width       int
	height      int
	msgCh       chan tea.Msg
	quitting    bool
	frame       int
	tickCount   int
	stats       PipelineStats
	events      []EventLine

	watchCfg      *WatchConfig
	watchCfgPath  string
	cfgFocus      int
	cfgCursors    [3]int
	cfgAdding     bool
	cfgConfirmDel bool
	cfgInput      string
	cfgErr        string
}

func NewModel() Model {
	m := Model{
		tabs:     []string{"Overview", "Logs", "Services", "Charts", "Config"},
		services: defaultServices(),
		selected: 0,
		msgCh:    make(chan tea.Msg, 256),
	}

	for _, base := range []string{"../../scout-go", "services/scout-go", "scout-go"} {
		p := watchConfigPath(base)
		if cfg, err := loadRepoConfig(p); err == nil {
			m.watchCfg = cfg
			m.watchCfgPath = p
			break
		}
	}
	if m.watchCfg == nil {
		m.watchCfg = &WatchConfig{Repos: []string{}, Contracts: []string{}, Wallets: []string{}, PollIntervalSeconds: 60}
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.listenCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) listenCmd() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgCh
	}
}

func (m *Model) addEvent(svcName, text, kind string) {
	e := EventLine{
		Time:    time.Now().Format("15:04:05"),
		Service: svcName,
		Text:    text,
		Kind:    kind,
	}
	m.events = append(m.events, e)
	if len(m.events) > 50 {
		m.events = m.events[len(m.events)-50:]
	}
}
