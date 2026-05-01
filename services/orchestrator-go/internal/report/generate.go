package report

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"text/template"

	"github.com/local/swarm/orchestrator/internal/llm"
	"github.com/local/swarm/orchestrator/pkg/messages"
)

func Generate(ctx context.Context, client llm.Client, exploit messages.VerifiedExploit) (teaserPath, fullPath string, err error) {
	tmpl, err := template.ParseFiles("../../prompts/report_v1.md")
	if err != nil {
		return "", "", err
	}
	var buf bytes.Buffer
	data := map[string]any{
		"ExploitJSON":     exploit,
		"Severity":        "critical", // from finding
		"ContractAddress": "",
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", err
	}

	raw, err := client.Complete(ctx, "", buf.String())
	if err != nil {
		return "", "", err
	}

	parts := bytes.Split([]byte(raw), []byte("---"))
	var teaser, full string
	for i, p := range parts {
		s := bytes.TrimSpace(p)
		if bytes.HasPrefix(s, []byte("TEASER:")) {
			teaser = string(bytes.TrimSpace(bytes.TrimPrefix(s, []byte("TEASER:"))))
		} else if bytes.HasPrefix(s, []byte("FULL:")) {
			full = string(bytes.TrimSpace(bytes.TrimPrefix(s, []byte("FULL:"))))
		} else if i == 1 && teaser == "" {
			teaser = string(s)
		} else if i == 2 && full == "" {
			full = string(s)
		}
	}

	outDir := filepath.Join("/tmp/orchestrator/reports", exploit.ID)
	_ = os.MkdirAll(outDir, 0755)
	teaserPath = filepath.Join(outDir, "teaser.md")
	fullPath = filepath.Join(outDir, "full.md")
	_ = os.WriteFile(teaserPath, []byte(teaser), 0644)
	_ = os.WriteFile(fullPath, []byte(full), 0644)
	return teaserPath, fullPath, nil
}
