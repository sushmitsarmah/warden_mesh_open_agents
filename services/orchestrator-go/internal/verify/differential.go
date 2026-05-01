package verify

import "context"

func DifferentialCheck(ctx context.Context, findingID string) (bool, error) {
	// Placeholder: use LLM to patch contract, deploy on anvil, re-run exploit.
	_ = findingID
	return true, nil
}
