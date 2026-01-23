package kroki

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://kroki.io"
	defaultTimeout = 30 * time.Second
)

// Client handles communication with the Kroki API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Kroki client with default settings.
func NewClient() *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
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
