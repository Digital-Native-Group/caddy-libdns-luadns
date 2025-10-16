package luadns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.luadns.com/v1"
)

// Client is an HTTP client for the Lua DNS API
type Client struct {
	email      string
	apiKey     string
	httpClient *http.Client
}

// Zone represents a DNS zone in Lua DNS
type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Record represents a DNS record in Lua DNS
type Record struct {
	ID      int    `json:"id,omitempty"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	ZoneID  int    `json:"zone_id,omitempty"`
}

// APIError represents an error response from the Lua DNS API
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Lua DNS API error (status %d): %s", e.StatusCode, e.Message)
}

// NewClient creates a new Lua DNS API client
func NewClient(email, apiKey string) *Client {
	return &Client{
		email:  email,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set HTTP Basic Auth
	req.SetBasicAuth(c.email, c.apiKey)

	// Set headers
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	return resp, nil
}

// ListZones retrieves all zones from the Lua DNS API
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/zones", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var zones []Zone
	if err := json.NewDecoder(resp.Body).Decode(&zones); err != nil {
		return nil, fmt.Errorf("failed to decode zones response: %w", err)
	}

	return zones, nil
}

// ListRecords retrieves all records for a zone
func (c *Client) ListRecords(ctx context.Context, zoneID int) ([]Record, error) {
	path := fmt.Sprintf("/zones/%d/records", zoneID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var records []Record
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to decode records response: %w", err)
	}

	return records, nil
}

// CreateRecord creates a new DNS record
func (c *Client) CreateRecord(ctx context.Context, zoneID int, record Record) (Record, error) {
	path := fmt.Sprintf("/zones/%d/records", zoneID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, record)
	if err != nil {
		return Record{}, err
	}
	defer resp.Body.Close()

	var created Record
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return Record{}, fmt.Errorf("failed to decode create response: %w", err)
	}

	return created, nil
}

// UpdateRecord updates an existing DNS record
func (c *Client) UpdateRecord(ctx context.Context, zoneID, recordID int, record Record) (Record, error) {
	path := fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID)
	resp, err := c.doRequest(ctx, http.MethodPut, path, record)
	if err != nil {
		return Record{}, err
	}
	defer resp.Body.Close()

	var updated Record
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return Record{}, fmt.Errorf("failed to decode update response: %w", err)
	}

	return updated, nil
}

// DeleteRecord deletes a DNS record
func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID int) error {
	path := fmt.Sprintf("/zones/%d/records/%d", zoneID, recordID)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
