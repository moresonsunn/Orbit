package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestContext bundles test client and cleanup.
type TestContext struct {
	Client   *Client
	IsLive   bool
	teardown func()
}

// SetupTest initializes client, auth session, and returns cleanup func.
func SetupTest(t *testing.T) (*Client, func()) {
	mode := os.Getenv("ORBIT_E2E_MODE")

	if mode == "contract" {
		cs := NewContractServer()
		client, err := NewClient(cs.URL())
		if err != nil {
			cs.Close()
			t.Fatalf("Failed to create client for contract server: %v", err)
		}
		if err := client.EnsureAuthenticated(); err != nil {
			cs.Close()
			t.Fatalf("Failed to authenticate with contract server: %v", err)
		}
		return client, func() { cs.Close() }
	}

	liveURL := os.Getenv("ORBIT_URL")
	if liveURL == "" {
		liveURL = "http://127.0.0.1:8787"
	}

	// Probe live server
	liveClient, err := NewClient(liveURL)
	isLiveReady := false

	if err == nil {
		probeHTTP := &http.Client{Timeout: 1 * time.Second}
		resp, err := probeHTTP.Get(liveURL + "/api/v1/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			// Probe if enriched endpoints are already live
			if mode == "live" {
				isLiveReady = true
			} else {
				// Check if /api/v1/netflow has containerTraffic or networks has SubnetDetails
				if err := liveClient.EnsureAuthenticated(); err == nil {
					netResp, netBody, err := liveClient.RawGet("/api/v1/networks")
					if err == nil && netResp.StatusCode == http.StatusOK {
						var rawNets []map[string]interface{}
						if json.Unmarshal(netBody, &rawNets) == nil && len(rawNets) > 0 {
							if _, hasSubnet := rawNets[0]["SubnetDetails"]; hasSubnet {
								isLiveReady = true
							}
						}
					}
				}
			}
		}
	}

	if isLiveReady {
		if err := liveClient.EnsureAuthenticated(); err != nil {
			t.Fatalf("Failed to authenticate with live Orbit server at %s: %v", liveURL, err)
		}
		return liveClient, func() {}
	}

	// Fallback to reference contract server
	cs := NewContractServer()
	client, err := NewClient(cs.URL())
	if err != nil {
		cs.Close()
		t.Fatalf("Failed to create client for contract server: %v", err)
	}
	if err := client.EnsureAuthenticated(); err != nil {
		cs.Close()
		t.Fatalf("Failed to authenticate with contract server: %v", err)
	}

	return client, func() { cs.Close() }
}
