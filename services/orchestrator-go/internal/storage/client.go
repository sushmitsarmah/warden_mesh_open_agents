package storage

import (
	"bytes"
	"context"
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
)

// LightClient uploads/downloads data via 0G Storage HTTP gateway.
// It avoids heavy SDK dependencies (web3go, EVM tx signing) by using
// the direct HTTP file upload API exposed by storage nodes.
type LightClient struct {
	gatewayURL string // e.g. "https://storage-turbo-testnet.0g.ai"
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

// NewLightClient creates a lightweight HTTP-only 0G Storage client.
func NewLightClient(gatewayURL string) *LightClient {
	if gatewayURL == "" {
		// Turbo testnet default
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

// Upload stores a byte slice in 0G Storage and returns its root hash.
// If no gateway is available it falls back to local filesystem (dev mode).
func (c *LightClient) Upload(ctx context.Context, name string, data []byte) (*UploadResult, error) {
	if c.gatewayURL == "" || isOfflineMode() {
		return c.uploadLocal(name, data)
	}

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

	slog.Info("0g.storage.upload", "name", name, "size", len(data), "gateway", c.gatewayURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Fallback to local on any gateway error
		slog.Warn("0g.storage.upload.gateway-failed, falling back to local", "status", resp.StatusCode, "body", string(respBody))
		return c.uploadLocal(name, data)
	}

	var result UploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse upload response: %w (body: %s)", err, string(respBody))
	}

	slog.Info("0g.storage.upload.done", "rootHash", result.RootHash, "size", result.Size)
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
