package tui

import (
	"bufio"
	"os"
	"os/exec"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Process struct {
	cmd     *exec.Cmd
	started time.Time
}

func startProcess(svc *Service, msgCh chan<- tea.Msg) {
	cmd := exec.Command(svc.Cmd[0], svc.Cmd[1:]...)
	cmd.Dir = svc.Dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		msgCh <- processLogMsg{Service: svc, Line: "failed to create stdout pipe: " + err.Error()}
		msgCh <- processExitMsg{Service: svc, Err: err}
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		msgCh <- processLogMsg{Service: svc, Line: "failed to create stderr pipe: " + err.Error()}
		msgCh <- processExitMsg{Service: svc, Err: err}
		return
	}

	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		msgCh <- processLogMsg{Service: svc, Line: "failed to start: " + err.Error()}
		msgCh <- processExitMsg{Service: svc, Err: err}
		return
	}

	p := &Process{cmd: cmd, started: time.Now()}
	svc.Process = p

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			msgCh <- processLogMsg{Service: svc, Line: scanner.Text()}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			msgCh <- processLogMsg{Service: svc, Line: "[stderr] " + scanner.Text()}
		}
	}()

	waitErr := cmd.Wait()
	msgCh <- processExitMsg{Service: svc, Err: waitErr}
}

func (p *Process) Stop() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
}
