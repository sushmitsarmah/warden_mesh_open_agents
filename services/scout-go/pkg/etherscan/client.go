package etherscan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client interacts with the Etherscan API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Etherscan API client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = "YourApiKeyToken" // Default placeholder for testing
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.etherscan.io/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SourceCodeResponse represents the Etherscan API response for contract source code
type SourceCodeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		SourceCode           string `json:"SourceCode"`
		ABI                  string `json:"ABI"`
		ContractName         string `json:"ContractName"`
		CompilerVersion      string `json:"CompilerVersion"`
		OptimizationUsed     string `json:"OptimizationUsed"`
		Runs                 string `json:"Runs"`
		ConstructorArguments string `json:"ConstructorArguments"`
		EVMVersion           string `json:"EVMVersion"`
		Library              string `json:"Library"`
		LicenseType          string `json:"LicenseType"`
		Proxy                string `json:"Proxy"`
		Implementation       string `json:"Implementation"`
		SwarmSource          string `json:"SwarmSource"`
	} `json:"result"`
}

// ContractSource contains the parsed contract source code and metadata
type ContractSource struct {
	Address         string
	ContractName    string
	SourceCode      string
	CompilerVersion string
	IsVerified      bool
	IsProxy         bool
	Implementation  string
}

// GetContractSource fetches the verified source code for a contract address
func (c *Client) GetContractSource(address string) (*ContractSource, error) {
	url := fmt.Sprintf("%s?module=contract&action=getsourcecode&address=%s&apikey=%s",
		c.baseURL, address, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("etherscan API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp SourceCodeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse etherscan response: %w", err)
	}

	if apiResp.Status != "1" {
		return nil, fmt.Errorf("etherscan API error: %s", apiResp.Message)
	}

	if len(apiResp.Result) == 0 {
		return nil, fmt.Errorf("no source code found for address %s", address)
	}

	result := apiResp.Result[0]

	// Check if contract is verified
	if result.SourceCode == "" {
		return nil, fmt.Errorf("contract %s is not verified on Etherscan", address)
	}

	source := &ContractSource{
		Address:         address,
		ContractName:    result.ContractName,
		SourceCode:      result.SourceCode,
		CompilerVersion: result.CompilerVersion,
		IsVerified:      true,
		IsProxy:         result.Proxy == "1",
		Implementation:  result.Implementation,
	}

	return source, nil
}

// GetMultiFileSource handles multi-file contracts (returns map of filename -> source)
func (c *Client) GetMultiFileSource(address string) (map[string]string, error) {
	source, err := c.GetContractSource(address)
	if err != nil {
		return nil, err
	}

	// Check if source is a JSON-encoded multi-file contract
	if len(source.SourceCode) > 0 && source.SourceCode[0] == '{' {
		var multiFile map[string]interface{}
		if err := json.Unmarshal([]byte(source.SourceCode), &multiFile); err != nil {
			// Not a multi-file contract, return single file
			return map[string]string{
				source.ContractName + ".sol": source.SourceCode,
			}, nil
		}

		// Extract sources from multi-file format
		files := make(map[string]string)
		if sources, ok := multiFile["sources"].(map[string]interface{}); ok {
			for filename, fileData := range sources {
				if fileMap, ok := fileData.(map[string]interface{}); ok {
					if content, ok := fileMap["content"].(string); ok {
						files[filename] = content
					}
				}
			}
		}

		if len(files) > 0 {
			return files, nil
		}
	}

	// Single file contract
	return map[string]string{
		source.ContractName + ".sol": source.SourceCode,
	}, nil
}
