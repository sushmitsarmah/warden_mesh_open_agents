package safety

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

type AuditLog struct {
	mu   sync.Mutex
	file *os.File

	lineCount int
}

func NewAuditLog(path string) (*AuditLog, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &AuditLog{file: f}, nil
}

func (a *AuditLog) Log(event string, fields map[string]any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339),
		"event": event,
		"data":  fields,
	}
	b, _ := json.Marshal(entry)
	if _, err := a.file.Write(append(b, '\n')); err != nil {
		slog.Error("audit write", "err", err)
	}
	a.lineCount++
	if a.lineCount%100 == 0 {
		a.writeHashCheckpoint()
	}
}

func (a *AuditLog) writeHashCheckpoint() {
	// Placeholder: compute memory hash and write checkpoint
	_, _ = a.file.WriteString(fmt.Sprintf(`%s
`, "checkpoint"))
}
