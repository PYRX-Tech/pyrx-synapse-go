package synapse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://synapse-api.pyrx.tech"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	maxBackoffSec     = 30.0
	jitterMaxSec      = 0.5
)

var retryableStatuses = map[int]bool{
	429: true,
	500: true,
	502: true,
	503: true,
	504: true,
}

// Client is the main Synapse API client.
type Client struct {
	apiKey      string
	workspaceID string
	baseURL     string
	timeout     time.Duration
	maxRetries  int
	httpClient  *http.Client
	Environment string
	Contacts    *ContactsClient
	Templates   *TemplatesClient

	// sleepFunc is used internally for testing (to avoid real sleeps).
	sleepFunc func(time.Duration)
}

// NewClient creates a new Synapse client with the given configuration.
func NewClient(config Config) (*Client, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("Synapse: api_key is required")
	}
	if strings.TrimSpace(config.WorkspaceID) == "" {
		return nil, fmt.Errorf("Synapse: workspace_id is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	maxRetries := config.MaxRetries
	if maxRetries == 0 && config.MaxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	c := &Client{
		apiKey:      config.APIKey,
		workspaceID: config.WorkspaceID,
		baseURL:     baseURL,
		timeout:     timeout,
		maxRetries:  maxRetries,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		Environment: detectEnvironment(config.APIKey),
		sleepFunc:   time.Sleep,
	}

	c.Contacts = &ContactsClient{client: c}
	c.Templates = &TemplatesClient{client: c}

	return c, nil
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Track tracks a single event.
func (c *Client) Track(params TrackParams) (*TrackResponse, error) {
	var resp TrackResponse
	if err := c.request("POST", "/v1/events", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// TrackBatch tracks multiple events in a single request.
func (c *Client) TrackBatch(params TrackBatchParams) (*BatchTrackResponse, error) {
	var resp BatchTrackResponse
	if err := c.request("POST", "/v1/events/batch", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Identify creates or updates a contact.
func (c *Client) Identify(params IdentifyParams) (*ContactResponse, error) {
	var resp ContactResponse
	if err := c.request("POST", "/v1/contacts", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IdentifyBatch creates or updates multiple contacts.
func (c *Client) IdentifyBatch(params IdentifyBatchParams) (*BulkContactResponse, error) {
	var resp BulkContactResponse
	if err := c.request("POST", "/v1/contacts/bulk", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Send sends an email using a template.
func (c *Client) Send(params SendParams) (*SendResponse, error) {
	var resp SendResponse
	if err := c.request("POST", "/v1/send", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// request performs an HTTP request with retry logic.
// If result is nil, the response body is discarded (used for 204 responses).
func (c *Client) request(method, path string, body interface{}, result interface{}) error {
	return c.requestWithParams(method, path, body, nil, result)
}

// requestWithParams performs an HTTP request with query parameters and retry logic.
func (c *Client) requestWithParams(method, path string, body interface{}, params map[string]string, result interface{}) error {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		fullURL += "?" + q.Encode()
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequest(method, fullURL, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Workspace-Id", c.workspaceID)
		req.Header.Set("X-Api-Key", c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "pyrx-synapse-go/"+Version)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt >= c.maxRetries {
				break
			}
			c.sleep(backoffDelay(attempt, nil))
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			if attempt >= c.maxRetries {
				break
			}
			c.sleep(backoffDelay(attempt, nil))
			continue
		}

		if resp.StatusCode == 204 {
			return nil
		}

		var parsed map[string]interface{}
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &parsed); err != nil {
				// Try parsing as array (for template list)
				var arr []interface{}
				if err2 := json.Unmarshal(respBody, &arr); err2 == nil {
					// For array responses, unmarshal directly into result
					if result != nil {
						return json.Unmarshal(respBody, result)
					}
					return nil
				}
				parsed = map[string]interface{}{}
			}
		}

		if resp.StatusCode >= 400 {
			var retryAfter float64
			if raHeader := resp.Header.Get("Retry-After"); raHeader != "" {
				retryAfter, _ = strconv.ParseFloat(raHeader, 64)
			}

			apiErr := MapError(resp.StatusCode, parsed, retryAfter)
			lastErr = apiErr

			if !retryableStatuses[resp.StatusCode] || attempt >= c.maxRetries {
				return apiErr
			}

			c.sleep(backoffDelay(attempt, apiErr))
			continue
		}

		// Success
		if result != nil {
			return json.Unmarshal(respBody, result)
		}
		return nil
	}

	return lastErr
}

func (c *Client) sleep(d time.Duration) {
	if c.sleepFunc != nil {
		c.sleepFunc(d)
	}
}

func detectEnvironment(apiKey string) string {
	if strings.HasPrefix(apiKey, "psk_test_") {
		return "test"
	}
	if strings.HasPrefix(apiKey, "psk_live_") {
		return "live"
	}
	return "unknown"
}

func backoffDelay(attempt int, err error) time.Duration {
	if rle, ok := err.(*SynapseRateLimitError); ok && rle.RetryAfter > 0 {
		secs := rle.RetryAfter + rand.Float64()*jitterMaxSec
		return time.Duration(secs * float64(time.Second))
	}

	exponential := 1.0 * math.Pow(2, float64(attempt))
	capped := math.Min(exponential, maxBackoffSec)
	secs := capped + rand.Float64()*jitterMaxSec
	return time.Duration(secs * float64(time.Second))
}
