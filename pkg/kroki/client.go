package kroki

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://kroki.io"
	DefaultTimeout = 30 * time.Second
)

// Options configures the Kroki client.
type Options struct {
	BaseURL string
	Timeout time.Duration
}

// Client handles communication with the Kroki API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Kroki client with default settings.
func NewClient() *Client {
	return NewClientWithOptions(Options{})
}

// NewClientWithOptions creates a new Kroki client with explicit configuration.
func NewClientWithOptions(opts Options) *Client {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GenerateSVG sends a D2 diagram to Kroki and returns the SVG image data.
func (c *Client) GenerateSVG(diagram string) ([]byte, error) {
	url := fmt.Sprintf("%s/d2/svg", c.baseURL)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(diagram))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to Kroki: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kroki returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}
