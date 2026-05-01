package verify

import (
	"fmt"
	"strconv"
	"strings"
)

func ExtractDrainUsd(logs []string) (float64, error) {
	for _, line := range logs {
		if strings.Contains(line, "DRAIN_USD") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "DRAIN_USD" && i+1 < len(parts) {
					v, err := strconv.ParseFloat(parts[i+1], 64)
					if err == nil && v > 0 {
						return v, nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("DRAIN_USD not found in logs")
}
