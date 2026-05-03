package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/0gfoundation/0g-storage-client/common/blockchain"
	"github.com/0gfoundation/0g-storage-client/core"
	"github.com/0gfoundation/0g-storage-client/indexer"
	"github.com/0gfoundation/0g-storage-client/transfer"
)

// LightClient uploads/downloads data via 0G Storage.
// It uses the official 0G Go SDK when OG_STORAGE_INDEXER_URL is configured,
// falls back to the legacy HTTP gateway, and finally to local filesystem.
type LightClient struct {
	gatewayURL string // legacy HTTP gateway (e.g. "https://storage-turbo-testnet.0g.ai")
	httpClient *http.Client
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	RootHash   string `json:"rootHash"`   // Merkle root / file ID
	TxSequence uint64 `json:"txSequence"` // on-chain sequence number
	Size       int64  `json:"size"`       // bytes written
}

// FileInfo returned by the storage scan API.
type FileInfo struct {
	RootHash   string `json:"rootHash"`
	TxSequence uint64 `json:"txSequence"`
	Size       int64  `json:"size"`
	Finalized  bool   `json:"finalized"`
}

// NewLightClient creates a lightweight 0G Storage client.
func NewLightClient(gatewayURL string) *LightClient {
	if gatewayURL == "" {
		gatewayURL = "https://storage-turbo-testnet.0g.ai"
	}
	return &LightClient{
		gatewayURL: gatewayURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SetTimeout adjusts the HTTP timeout.
func (c *LightClient) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

// SelfTest uploads a 1KB test blob at startup to verify SDK connectivity.
// On failure it logs a warning but does NOT crash — the orchestrator keeps running.
func (c *LightClient) SelfTest(ctx context.Context) {
	if !hasSDKConfig() {
		slog.Info("0g storage self-test skipped: no SDK config (OG_STORAGE_INDEXER_URL)")
		return
	}
	testData := make([]byte, 1024)
	_, _ = rand.Read(testData)
	_, err := c.uploadWithSDK(ctx, "self-test", testData)
	if err != nil {
		slog.Warn("0g storage degraded at startup", "err", err)
	} else {
		slog.Info("0g storage self-test passed")
	}
}

// Upload stores a byte slice in 0G Storage and returns its root hash.
// Priority: 1) official SDK, 2) legacy HTTP gateway, 3) local filesystem fallback.
func (c *LightClient) Upload(ctx context.Context, name string, data []byte) (*UploadResult, error) {
	// 1. Try official SDK if configured.
	if hasSDKConfig() {
		result, err := c.uploadWithSDK(ctx, name, data)
		if err == nil {
			return result, nil
		}
		slog.Warn("0g storage degraded, fell back to local", "err", err)
	}

	// 2. Try legacy HTTP gateway.
	if c.gatewayURL != "" && !isOfflineMode() {
		result, err := c.uploadHTTP(ctx, name, data)
		if err == nil {
			return result, nil
		}
		slog.Warn("0g storage http gateway failed, fell back to local", "err", err)
	}

	// 3. Final fallback: local filesystem.
	return c.uploadLocal(name, data)
}

// uploadWithSDK uses the official 0G Storage Go SDK to upload data.
// NOTE: The SDK (v1.3.0) can panic inside background goroutines when a storage node
// hasn't synced the on-chain log entry. Until 0G fixes this upstream, the SDK path
// is gated behind OG_STORAGE_USE_SDK=true.
func (c *LightClient) uploadWithSDK(ctx context.Context, name string, data []byte) (result *UploadResult, err error) {
	rpcURL := os.Getenv("OG_STORAGE_RPC_URL")
	if rpcURL == "" {
		rpcURL = os.Getenv("OG_RPC_URL")
	}
	indexerURL := os.Getenv("OG_STORAGE_INDEXER_URL")
	privateKey := os.Getenv("OG_PRIVATE_KEY")

	if rpcURL == "" {
		return nil, fmt.Errorf("OG_STORAGE_RPC_URL or OG_RPC_URL required")
	}
	if indexerURL == "" {
		return nil, fmt.Errorf("OG_STORAGE_INDEXER_URL required")
	}
	if privateKey == "" {
		return nil, fmt.Errorf("OG_PRIVATE_KEY required for signing storage uploads")
	}

	// Apply a hard deadline so a hanging node sync doesn't block forever.
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// Create Web3 client for on-chain interactions.
	w3Client, err := blockchain.NewWeb3(rpcURL, privateKey)
	if err != nil {
		return nil, fmt.Errorf("create web3 client: %w", err)
	}
	defer w3Client.Close()

	// Create indexer client for node discovery.
	indexerClient, err := indexer.NewClient(indexerURL, indexer.IndexerClientOption{})
	if err != nil {
		return nil, fmt.Errorf("create indexer client: %w", err)
	}

	// Prepare in-memory data.
	dataInMem, err := core.NewDataInMemory(data)
	if err != nil {
		return nil, fmt.Errorf("create in-memory data: %w", err)
	}

	fragmentSize := int64(4 * 1024 * 1024 * 1024) // 4GB
	opt := transfer.UploadOption{
		ExpectedReplica:  1,
		TaskSize:         10,
		SkipTx:           true,
		FinalityRequired: transfer.FileFinalized,
		FastMode:         false,
		Method:           "min",
		FullTrusted:      true,
	}

	slog.Info("0g.storage.upload.sdk", "name", name, "size", len(data), "indexer", indexerURL)

	txHashes, roots, err := indexerClient.SplitableUpload(ctx, w3Client, dataInMem, fragmentSize, opt)
	if err != nil {
		return nil, fmt.Errorf("sdk upload failed: %w", err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no roots returned from upload")
	}

	rootHash := roots[0].Hex()
	slog.Info("0g.storage.upload.sdk.done",
		"rootHash", rootHash,
		"txHashes", len(txHashes),
		"size", len(data),
	)

	return &UploadResult{
		RootHash: rootHash,
		Size:     int64(len(data)),
	}, nil
}

// uploadHTTP uses the legacy HTTP gateway endpoint.
func (c *LightClient) uploadHTTP(ctx context.Context, name string, data []byte) (*UploadResult, error) {
	url := c.gatewayURL + "/upload"

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	slog.Info("0g.storage.upload.http", "name", name, "size", len(data), "gateway", c.gatewayURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result UploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse upload response: %w (body: %s)", err, string(respBody))
	}

	slog.Info("0g.storage.upload.http.done", "rootHash", result.RootHash, "size", result.Size)
	return &result, nil
}

// UploadJSON is a convenience wrapper that stores a JSON-marshalable value.
func (c *LightClient) UploadJSON(ctx context.Context, name string, v any) (*UploadResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return c.Upload(ctx, name, data)
}

// Download fetches data by root hash.
func (c *LightClient) Download(ctx context.Context, rootHash string) ([]byte, error) {
	if c.gatewayURL == "" || isOfflineMode() {
		return c.downloadLocal(rootHash)
	}

	url := fmt.Sprintf("%s/download?rootHash=%s", c.gatewayURL, rootHash)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// DownloadJSON downloads and unmarshals JSON.
func (c *LightClient) DownloadJSON(ctx context.Context, rootHash string, v any) error {
	data, err := c.Download(ctx, rootHash)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// FileInfo queries the storage scan API for file status.
func (c *LightClient) FileInfo(ctx context.Context, rootHash string) (*FileInfo, error) {
	scanURL := "https://storagescan-testnet.0g.ai/api/files/" + rootHash
	req, err := http.NewRequestWithContext(ctx, "GET", scanURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storagescan returned %d", resp.StatusCode)
	}

	var info FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ── Helper ──────────────────────────────────────────────────────────

func hasSDKConfig() bool {
	// SDK is opt-in because v1.3.0 can panic inside background goroutines when
	// storage nodes haven't synced the on-chain log entry yet.
	// To enable: set OG_STORAGE_USE_SDK=true along with the indexer + wallet config.
	return os.Getenv("OG_STORAGE_USE_SDK") == "true" &&
		os.Getenv("OG_STORAGE_INDEXER_URL") != "" &&
		os.Getenv("OG_PRIVATE_KEY") != ""
}

// ── Offline fallback (dev mode) ─────────────────────────────────────

const localStoreDir = "/tmp/0g-storage-local"

func isOfflineMode() bool {
	return os.Getenv("0G_STORAGE_OFFLINE") == "true" || os.Getenv("STORAGE_GATEWAY_URL") == ""
}

func (c *LightClient) uploadLocal(name string, data []byte) (*UploadResult, error) {
	root := localRootHash(data)
	_ = os.MkdirAll(localStoreDir, 0755)
	path := fmt.Sprintf("%s/%s_%s", localStoreDir, root, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}
	slog.Info("0g.storage.upload.local", "path", path, "rootHash", root, "size", len(data))
	return &UploadResult{RootHash: root, Size: int64(len(data))}, nil
}

func (c *LightClient) downloadLocal(rootHash string) ([]byte, error) {
	matches, err := os.ReadDir(localStoreDir)
	if err != nil {
		return nil, err
	}
	for _, e := range matches {
		if e.IsDir() {
			continue
		}
		if bytes.HasPrefix([]byte(e.Name()), []byte(rootHash)) {
			return os.ReadFile(localStoreDir + "/" + e.Name())
		}
	}
	return nil, fmt.Errorf("local file not found for root %s", rootHash)
}

func localRootHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
