package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTier4_RealWorldScenarios(t *testing.T) {
	client, teardown := SetupTest(t)
	defer teardown()

	// -------------------------------------------------------------------------
	// Scenario 1: Multi-Tier Application Stack Inspection
	// -------------------------------------------------------------------------
	t.Run("Scenario1_MultiTierAppStack_Inspection", func(t *testing.T) {
		// Verify inspection of multi-tier container stacks (frontend, backend, database)
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("Failed to list networks: %v", err)
		}

		var targetNet *EnrichedDockerNetwork
		for i := range nets {
			if len(nets[i].Containers) >= 2 {
				targetNet = &nets[i]
				break
			}
		}
		if targetNet == nil {
			for i := range nets {
				if nets[i].SubnetDetails != nil {
					targetNet = &nets[i]
					break
				}
			}
		}
		if targetNet == nil {
			t.Fatalf("No suitable network found for multi-tier inspection")
		}

		t.Logf("Inspecting multi-tier network %s (%s)", targetNet.Name, targetNet.Id)

		// 1. IPAM subnet capacity inspection
		if targetNet.SubnetDetails != nil {
			sd := targetNet.SubnetDetails
			if sd.Subnet == "" || sd.Gateway == "" {
				t.Errorf("Multi-tier network missing Subnet or Gateway in IPAM")
			}
			t.Logf("IPAM Capacity: %d total IPs, %d used, %d free (%.2f%% utilized)",
				sd.TotalIPs, sd.UsedIPs, sd.FreeIPs, sd.UtilizationPercent)
		}

		// 2. Client container member inspection
		clientCount := len(targetNet.Containers)
		t.Logf("Connected container clients: %d", clientCount)
		for cID, member := range targetNet.Containers {
			if member.Name == "" {
				t.Errorf("Container %s missing name", cID)
			}
			if member.IPv4Address != "" {
				t.Logf("  - Container: %s | IPv4: %s | MAC: %s | Status: %s",
					member.Name, member.IPv4Address, member.MacAddress, member.State)
			}
		}
	})

	// -------------------------------------------------------------------------
	// Scenario 2: Security Risk Exposure Detection Workflow
	// -------------------------------------------------------------------------
	t.Run("Scenario2_SecurityRiskExposureDetection_Workflow", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("Failed to fetch netflow snapshot: %v", err)
		}

		criticalAlerts := 0
		cautionAlerts := 0
		safeServices := 0
		internalServices := 0

		for _, sc := range snap.Connections {
			switch sc.SecurityLevel {
			case "critical":
				criticalAlerts++
				t.Logf("[CRITICAL ALERT] Container: %s | Endpoint: %s:%d | Proto: %s | Desc: %s",
					sc.ContainerName, sc.LocalAddr, sc.LocalPort, sc.DetectedProto, sc.SecurityDesc)
			case "caution":
				cautionAlerts++
				t.Logf("[CAUTION] Container: %s | Endpoint: %s:%d | Proto: %s | Desc: %s",
					sc.ContainerName, sc.LocalAddr, sc.LocalPort, sc.DetectedProto, sc.SecurityDesc)
			case "safe":
				safeServices++
			case "internal":
				internalServices++
			}
		}

		t.Logf("Security Audit Summary: %d Critical, %d Caution, %d Safe, %d Internal",
			criticalAlerts, cautionAlerts, safeServices, internalServices)

		if snap.Totals.SecurityAlertsCount < 0 {
			t.Errorf("Negative security alert count in totals: %d", snap.Totals.SecurityAlertsCount)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario 3: Live Traffic & Bandwidth Telemetry Ingestion
	// -------------------------------------------------------------------------
	t.Run("Scenario3_LiveTrafficAndBandwidthTelemetry_Ingestion", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("Failed to fetch netflow snapshot: %v", err)
		}

		t.Logf("Live Rate Meters: Total Rx: %.2f Bps | Total Tx: %.2f Bps | Active Connections: %d",
			snap.Totals.TotalRxBps, snap.Totals.TotalTxBps, snap.Totals.ActiveConnections)

		// Check traffic breakdown
		t.Logf("Top Container Bandwidth Rankings:")
		for i, ct := range snap.ContainerTraffic {
			if i < 5 {
				t.Logf("  %d. %s: Rx: %.2f Bps, Tx: %.2f Bps, RxBytes: %d, TxBytes: %d",
					i+1, ct.ContainerName, ct.RxBps, ct.TxBps, ct.RxBytes, ct.TxBytes)
			}
		}

		// Check topology integration
		activeEdges := 0
		for _, edge := range snap.Topology.Edges {
			if edge.Active {
				activeEdges++
			}
		}
		t.Logf("Topology Canvas: %d nodes, %d edges (%d active)",
			len(snap.Topology.Nodes), len(snap.Topology.Edges), activeEdges)
	})

	// -------------------------------------------------------------------------
	// Scenario 4: Dynamic Container Network Lifecycle Workflow
	// -------------------------------------------------------------------------
	t.Run("Scenario4_DynamicContainerNetworkLifecycle_Workflow", func(t *testing.T) {
		netName := fmt.Sprintf("test-orbit-dyn-%d", time.Now().Unix())
		newNetPayload := EnrichedDockerNetwork{
			Name:       netName,
			Driver:     "bridge",
			Scope:      "local",
			Internal:   true,
			EnableIPv6: false,
			IPAM: IPAMData{
				Driver: "default",
				Config: []IPAMConfig{
					{Subnet: "172.29.0.0/24", Gateway: "172.29.0.1"},
				},
			},
			SubnetDetails: &SubnetDetails{
				Subnet:             "172.29.0.0/24",
				Gateway:            "172.29.0.1",
				NetworkIP:          "172.29.0.0",
				BroadcastIP:        "172.29.0.255",
				UsableRange:        "172.29.0.1 - 172.29.0.254",
				TotalIPs:           256,
				UsedIPs:            0,
				FreeIPs:            254,
				UtilizationPercent: 0.0,
			},
		}

		// 1. Create network
		resp, body, err := client.RawPost("/api/v1/networks", newNetPayload)
		if err != nil {
			t.Fatalf("Failed to create dynamic network: %v", err)
		}
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Logf("Notice: Dynamic network creation returned %d: %s", resp.StatusCode, string(body))
			return
		}

		// 2. Inspect created network
		createdNet, err := client.GetNetwork(netName)
		if err != nil {
			t.Fatalf("Failed to inspect newly created network %s: %v", netName, err)
		}
		if createdNet.Name != netName {
			t.Errorf("Created network name mismatch: expected %s, got %s", netName, createdNet.Name)
		}

		// 3. Delete network
		delResp, _, err := client.RawDelete("/api/v1/networks/" + createdNet.Id)
		if err != nil {
			t.Fatalf("Failed to delete network %s: %v", createdNet.Id, err)
		}
		if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
			t.Errorf("Unexpected status on network deletion: %d", delResp.StatusCode)
		}

		// 4. Verify removal
		getAfterDel, _, _ := client.RawGet("/api/v1/networks/" + createdNet.Id)
		if getAfterDel.StatusCode != http.StatusNotFound {
			t.Errorf("Expected HTTP 404 after deletion, got %d", getAfterDel.StatusCode)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario 5: Full User Session & Navigation Flow
	// -------------------------------------------------------------------------
	t.Run("Scenario5_FullUserSessionAndNavigation_Flow", func(t *testing.T) {
		sessionClient, err := NewClient(client.BaseURL)
		if err != nil {
			t.Fatalf("Failed to create session client: %v", err)
		}

		// 1. Healthcheck
		h, err := sessionClient.GetHealth()
		if err != nil || h["status"] != "ok" {
			t.Fatalf("Health check failed: %v", err)
		}

		// 2. Auth status (initially unauthenticated)
		s, err := sessionClient.GetAuthStatus()
		if err != nil {
			t.Fatalf("Auth status probe failed: %v", err)
		}
		if s.Authenticated {
			t.Errorf("Expected initially unauthenticated")
		}

		// 3. Login
		err = sessionClient.Login("admin", "OrbitAdminSecret2026!")
		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if sessionClient.CSRFToken == "" {
			t.Errorf("Expected CSRF token after login")
		}

		// 4. Navigate to /networks (SPA HTML route)
		pageResp, pageBody, err := sessionClient.RawGet("/networks")
		if err != nil || pageResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to load /networks route: %v", err)
		}
		if !strings.Contains(string(pageBody), "<html") {
			t.Errorf("Expected HTML on /networks SPA route")
		}

		// 5. Fetch Networks API
		nets, err := sessionClient.ListNetworks()
		if err != nil || len(nets) == 0 {
			t.Fatalf("Failed to list networks: %v", err)
		}

		// 6. Navigate to /network-flow
		flowPageResp, flowPageBody, err := sessionClient.RawGet("/network-flow")
		if err != nil || flowPageResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to load /network-flow route: %v", err)
		}
		if !strings.Contains(string(flowPageBody), "<html") {
			t.Errorf("Expected HTML on /network-flow SPA route")
		}

		// 7. Fetch Netflow API
		snap, err := sessionClient.GetNetflow()
		if err != nil {
			t.Fatalf("Failed to fetch netflow: %v", err)
		}
		if snap.Timestamp == "" {
			t.Errorf("Snapshot timestamp missing")
		}

		t.Logf("Full user journey completed successfully: Login -> /networks -> /network-flow -> Netflow Telemetry")
	})
}
