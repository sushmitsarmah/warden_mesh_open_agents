package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type TestResult struct {
	Passed  bool
	GasUsed uint64
	Logs    []string
}

func RunForkTest(exploitPath, forkURL string, forkBlock int64) (TestResult, error) {
	// copy exploit into forge-harness
	harness := "../../infra/forge-harness"
	dest := filepath.Join(harness, "test", filepath.Base(exploitPath))
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	b, _ := os.ReadFile(exploitPath)
	_ = os.WriteFile(dest, b, 0644)

	args := []string{"test", "--match-contract", "ExploitTest", "--json", "--fork-url", forkURL}
	if forkBlock > 0 {
		args = append(args, "--fork-block-number", strconv.FormatInt(forkBlock, 10))
	}
	cmd := exec.Command("forge", args...)
	cmd.Dir = harness
	out, err := cmd.CombinedOutput()
	if err != nil {
		return TestResult{}, fmt.Errorf("forge test failed: %w\n%s", err, string(out))
	}
	return parseForgeOutput(string(out))
}

func parseForgeOutput(raw string) (TestResult, error) {
	// naive parse: look for {"test_results":...}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"test_results"`) {
			var result map[string]any
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				return TestResult{Passed: true}, nil
			}
		}
	}
	return TestResult{Passed: false}, nil
}
