package inft

import "context"

// Recorder is the minimal interface for recording disclosures on-chain.
type Recorder interface {
	RecordDisclosure(ctx context.Context, tokenID, bountyUsd int64, memoryDelta [32]byte) error
	IsPaused(tokenID int64) (bool, error)
	IsAuthorized(protocol string) (bool, error)
}
