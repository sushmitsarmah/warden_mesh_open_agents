package messages

import "time"

type Target struct {
	ID           string    `json:"id"`
	BountyType   string    `json:"bountyType,omitempty"`
	Kind         string    `json:"kind"`
	ChainID      int       `json:"chainId,omitempty"`
	Address      string    `json:"address,omitempty"`
	Repo         string    `json:"repo,omitempty"`
	CommitSha    string    `json:"commitSha,omitempty"`
	SourceURL    string    `json:"sourceUrl,omitempty"`
	TxHash       string    `json:"txHash,omitempty"`
	DiscoveredAt time.Time `json:"discoveredAt"`
	Priority     float64   `json:"priority"`
	TVLUsd       float64   `json:"tvlUsd,omitempty"`
}

type Severity string

const (
	SevInfo     Severity = "info"
	SevLow      Severity = "low"
	SevMedium   Severity = "medium"
	SevHigh     Severity = "high"
	SevCritical Severity = "critical"
)

type Finding struct {
	ID          string   `json:"id"`
	TargetID    string   `json:"targetId"`
	BountyType  string   `json:"bountyType,omitempty"`
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	Tools       []string `json:"tools"`
	Location    Location `json:"location,omitempty"`
	Description string   `json:"description"`
}

type Location struct {
	File      string `json:"file"`
	LineStart int    `json:"lineStart"`
	LineEnd   int    `json:"lineEnd"`
}

type VerifiedExploit struct {
	ID                    string    `json:"id"`
	FindingID             string    `json:"findingId"`
	PoCPath               string    `json:"pocPath"`
	ImpactType            string    `json:"impactType"`
	ValidatedAgainstConfig bool     `json:"validatedAgainstConfig"`
	AttackerModel         string    `json:"attackerModel"`
	ModifiedValidator     bool      `json:"modifiedValidator"`
	Justification         string    `json:"justification,omitempty"`
	VerifiedAt            time.Time `json:"verifiedAt"`
}

type Disclosure struct {
	ID          string    `json:"id"`
	ExploitID   string    `json:"exploitId"`
	Lane        string    `json:"lane"`
	X402URL     string    `json:"x402Url,omitempty"`
	TxHash      string    `json:"txHash,omitempty"`
	StorageHash string    `json:"storageHash,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
}
