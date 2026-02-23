package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SecDuckOps/shared/protocol"
)

type DuckOpsClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *DuckOpsClient {
	return &DuckOpsClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// SubmitResult sends the scan results back to the cloud plane
func (c *DuckOpsClient) SubmitResult(res protocol.ScanResult) error {
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/results", c.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
