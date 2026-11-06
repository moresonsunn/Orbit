package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client is an opaque-box HTTP client for Orbit APIs.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CSRFToken  string
}

// NewClient initializes a new Orbit E2E client.
func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = os.Getenv("ORBIT_URL")
	}
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8787"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}, nil
}

func (c *Client) request(method, path string, body interface{}, out interface{}) (*http.Response, []byte, error) {
	targetURL := c.BaseURL + path
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.CSRFToken != "" && (method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH") {
		req.Header.Set("X-CSRF-Token", c.CSRFToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if out != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return resp, respBytes, fmt.Errorf("failed to decode JSON response: %w (body: %s)", err, string(respBytes))
		}
	}

	return resp, respBytes, nil
}

// GetHealth checks /api/v1/health.
func (c *Client) GetHealth() (map[string]interface{}, error) {
	var res map[string]interface{}
	resp, body, err := c.request("GET", "/api/v1/health", nil, &res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected health status %d: %s", resp.StatusCode, string(body))
	}
	return res, nil
}

// GetAuthStatus checks /api/v1/auth/status.
func (c *Client) GetAuthStatus() (*AuthStatus, error) {
	var res AuthStatus
	resp, body, err := c.request("GET", "/api/v1/auth/status", nil, &res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected auth status %d: %s", resp.StatusCode, string(body))
	}
	return &res, nil
}

// Login authenticates with username and password.
func (c *Client) Login(username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	var loginResp LoginResponse
	resp, body, err := c.request("POST", "/api/v1/auth/login", payload, &loginResp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}
	if loginResp.CSRFToken != "" {
		c.CSRFToken = loginResp.CSRFToken
	}
	return nil
}

// EnsureAuthenticated ensures client is authenticated using default or environment credentials.
func (c *Client) EnsureAuthenticated() error {
	status, err := c.GetAuthStatus()
	if err != nil {
		return err
	}
	if status.Authenticated {
		if status.CSRFToken != "" {
			c.CSRFToken = status.CSRFToken
		}
		return nil
	}

	user := os.Getenv("ORBIT_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("ORBIT_PASS")
	if pass == "" {
		pass = "OrbitAdminSecret2026!"
	}

	return c.Login(user, pass)
}

// ListNetworks fetches all enriched Docker networks.
func (c *Client) ListNetworks() ([]EnrichedDockerNetwork, error) {
	var networks []EnrichedDockerNetwork
	resp, body, err := c.request("GET", "/api/v1/networks", nil, &networks)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list networks (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return networks, nil
}

// GetNetwork fetches an individual network by ID.
func (c *Client) GetNetwork(id string) (*EnrichedDockerNetwork, error) {
	var net EnrichedDockerNetwork
	resp, body, err := c.request("GET", "/api/v1/networks/"+url.PathEscape(id), nil, &net)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get network %s (HTTP %d): %s", id, resp.StatusCode, string(body))
	}
	return &net, nil
}

// GetNetflow fetches the enriched netflow snapshot.
func (c *Client) GetNetflow() (*EnrichedNetflowSnapshot, error) {
	var snapshot EnrichedNetflowSnapshot
	resp, body, err := c.request("GET", "/api/v1/netflow", nil, &snapshot)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get netflow (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return &snapshot, nil
}

// ListContainers fetches the container list.
func (c *Client) ListContainers() ([]ContainerSummary, error) {
	var containers []ContainerSummary
	resp, body, err := c.request("GET", "/api/v1/containers", nil, &containers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list containers (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return containers, nil
}

// RawGet issues a GET request returning raw response and body.
func (c *Client) RawGet(path string) (*http.Response, []byte, error) {
	return c.request("GET", path, nil, nil)
}

// RawPost issues a POST request returning raw response and body.
func (c *Client) RawPost(path string, body interface{}) (*http.Response, []byte, error) {
	return c.request("POST", path, body, nil)
}

// RawDelete issues a DELETE request returning raw response and body.
func (c *Client) RawDelete(path string) (*http.Response, []byte, error) {
	return c.request("DELETE", path, nil, nil)
}
