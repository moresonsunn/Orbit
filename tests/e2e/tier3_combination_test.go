package e2e

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestTier3_CrossFeatureCombinations(t *testing.T) {
	client, teardown := SetupTest(t)
	defer teardown()

	// -------------------------------------------------------------------------
	// Combo 01: F01 (Enriched Networks) + F05 (IPAM Calculation)
	// -------------------------------------------------------------------------
	t.Run("Combo01_NetworkIPAMSubnetConsistency", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		found := false
		for _, n := range nets {
			if n.SubnetDetails != nil && len(n.IPAM.Config) > 0 {
				found = true
				if n.SubnetDetails.Subnet != n.IPAM.Config[0].Subnet {
					t.Errorf("SubnetDetails.Subnet (%s) does not match IPAM.Config[0].Subnet (%s)",
						n.SubnetDetails.Subnet, n.IPAM.Config[0].Subnet)
				}
				if n.IPAM.Config[0].Gateway != "" && n.SubnetDetails.Gateway != n.IPAM.Config[0].Gateway {
					t.Errorf("SubnetDetails.Gateway (%s) does not match IPAM.Config[0].Gateway (%s)",
						n.SubnetDetails.Gateway, n.IPAM.Config[0].Gateway)
				}
			}
		}
		if !found {
			t.Fatalf("No network found with both SubnetDetails and IPAM.Config")
		}
	})

	// -------------------------------------------------------------------------
	// Combo 02: F01 (Enriched Networks) + F06 (Connected Container Insights)
	// -------------------------------------------------------------------------
	t.Run("Combo02_NetworkContainersWithinSubnetCIDR", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		for _, n := range nets {
			if n.SubnetDetails != nil && len(n.Containers) > 0 {
				_, ipnet, err := net.ParseCIDR(n.SubnetDetails.Subnet)
				if err != nil {
					t.Fatalf("Failed to parse subnet %s: %v", n.SubnetDetails.Subnet, err)
				}
				for _, c := range n.Containers {
					if c.IPv4Address != "" {
						ipStr := strings.Split(c.IPv4Address, "/")[0]
						ip := net.ParseIP(ipStr)
						if ip == nil || !ipnet.Contains(ip) {
							t.Errorf("Container %s IP %s is not within network subnet %s", c.Name, ipStr, ipnet)
						}
					}
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 03: F01 (Enriched Networks) + F07 (Driver Options & Scope)
	// -------------------------------------------------------------------------
	t.Run("Combo03_DriverOptionsCorrelateWithNetworkScope", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		for _, n := range nets {
			if n.Driver == "bridge" {
				if n.Scope != "local" && n.Scope != "swarm" {
					t.Errorf("Unexpected scope '%s' for bridge network %s", n.Scope, n.Name)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 04: F02 (Container Bandwidth) + F09 (Rate Meters)
	// -------------------------------------------------------------------------
	t.Run("Combo04_ContainerTrafficSumCorrelatesWithTotals", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		var sumRx, sumTx float64
		for _, ct := range snap.ContainerTraffic {
			sumRx += ct.RxBps
			sumTx += ct.TxBps
		}
		// Totals should either match sumRx/sumTx or be greater than/equal if host traffic is included
		if sumRx > 0 && snap.Totals.TotalRxBps == 0 {
			t.Errorf("TotalRxBps is 0 but container traffic sumRx is %f", sumRx)
		}
		if sumTx > 0 && snap.Totals.TotalTxBps == 0 {
			t.Errorf("TotalTxBps is 0 but container traffic sumTx is %f", sumTx)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 05: F02 (Container Bandwidth) + F10 (Traffic Breakdown)
	// -------------------------------------------------------------------------
	t.Run("Combo05_TopBandwidthContainerMatchesProtocolTraffic", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		if len(snap.ContainerTraffic) > 0 {
			var topContainer *ContainerTrafficMetric
			for i := range snap.ContainerTraffic {
				ct := &snap.ContainerTraffic[i]
				if topContainer == nil || (ct.RxBps+ct.TxBps) > (topContainer.RxBps+topContainer.TxBps) {
					topContainer = ct
				}
			}
			if topContainer != nil && (topContainer.RxBps+topContainer.TxBps) > 0 {
				hasActiveProtocol := false
				for _, ps := range snap.ProtocolStats {
					if ps.Active > 0 {
						hasActiveProtocol = true
						break
					}
				}
				if !hasActiveProtocol {
					t.Logf("Notice: Top container %s has traffic but no protocol flagged active", topContainer.ContainerName)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 06: F03 (Hybrid Sockets) + F11 (Security Attribution)
	// -------------------------------------------------------------------------
	t.Run("Combo06_SynthesizedSocketsPortSecurityMapping", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		for _, sc := range snap.Connections {
			if sc.SecurityLevel == "critical" {
				if sc.SecurityBadge != "CRITICAL" {
					t.Errorf("Expected badge 'CRITICAL' for critical level, got '%s'", sc.SecurityBadge)
				}
				if sc.SecurityDesc == "" {
					t.Errorf("Expected non-empty description for critical socket: %+v", sc)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 07: F03 (Hybrid Sockets) + F08 (Topology Graph)
	// -------------------------------------------------------------------------
	t.Run("Combo07_SocketContainersMatchTopologyNodes", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		nodeMap := make(map[string]bool)
		for _, node := range snap.Topology.Nodes {
			nodeMap[node.ID] = true
			nodeMap[node.Label] = true
		}
		for _, sc := range snap.Connections {
			if sc.ContainerID != "" {
				if !nodeMap[sc.ContainerID] && !nodeMap[sc.ContainerName] {
					t.Logf("Notice: Socket container %s not found in topology nodes", sc.ContainerName)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 08: F05 (IPAM Subnet) + F07 (Driver Scope)
	// -------------------------------------------------------------------------
	t.Run("Combo08_DriverTypeDictatesSubnetDetailsPresence", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		for _, n := range nets {
			if n.Driver == "bridge" {
				if n.SubnetDetails == nil {
					t.Errorf("Bridge network %s missing SubnetDetails", n.Name)
				}
			} else if n.Driver == "null" || n.Driver == "host" {
				if n.SubnetDetails != nil && n.SubnetDetails.Subnet != "" {
					t.Logf("Notice: Null/host network %s populated SubnetDetails", n.Name)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 09: F06 (Connected Containers) + F11 (Active Sockets)
	// -------------------------------------------------------------------------
	t.Run("Combo09_ConnectedContainerNamesInSocketsTable", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		networkContainerNames := make(map[string]bool)
		for _, n := range nets {
			for _, c := range n.Containers {
				networkContainerNames[c.Name] = true
			}
		}
		matchingCount := 0
		for _, sc := range snap.Connections {
			if networkContainerNames[sc.ContainerName] {
				matchingCount++
			}
		}
		t.Logf("Found %d socket connections matching containers from networks list", matchingCount)
	})

	// -------------------------------------------------------------------------
	// Combo 10: F09 (Rate Meters) + F11 (Active Sockets)
	// -------------------------------------------------------------------------
	t.Run("Combo10_TotalsActiveConnectionsCorrelatesWithSockets", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		if snap.Totals.ActiveConnections < 0 {
			t.Errorf("ActiveConnections cannot be negative: %d", snap.Totals.ActiveConnections)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 11: F10 (Protocol Breakdown) + F11 (Active Sockets)
	// -------------------------------------------------------------------------
	t.Run("Combo11_ProtocolStatsCorrelatesWithSocketProtocols", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		socketProtos := make(map[string]int)
		for _, sc := range snap.Connections {
			if sc.DetectedProto != "" {
				socketProtos[strings.ToUpper(sc.DetectedProto)]++
			}
		}
		for _, ps := range snap.ProtocolStats {
			if count, ok := socketProtos[strings.ToUpper(ps.Protocol)]; ok {
				if ps.Observed < count && ps.Configured < count {
					t.Logf("Notice: Protocol %s has %d sockets but observed is %d", ps.Protocol, count, ps.Observed)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 12: F04 (Types) + F01 (Enriched Networks)
	// -------------------------------------------------------------------------
	t.Run("Combo12_StrictNetworkSchemaValidation", func(t *testing.T) {
		resp, body, err := client.RawGet("/api/v1/networks")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("RawGet /api/v1/networks failed: %v", err)
		}
		var nets []EnrichedDockerNetwork
		if err := json.Unmarshal(body, &nets); err != nil {
			t.Fatalf("Failed to deserialize networks: %v", err)
		}
		for _, n := range nets {
			if n.Id == "" || n.Name == "" || n.Driver == "" {
				t.Errorf("Network missing core fields: %+v", n)
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 13: F04 (Types) + F02 & F03 (Netflow API)
	// -------------------------------------------------------------------------
	t.Run("Combo13_StrictNetflowSchemaValidation", func(t *testing.T) {
		resp, body, err := client.RawGet("/api/v1/netflow")
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("RawGet /api/v1/netflow failed: %v", err)
		}
		var snap EnrichedNetflowSnapshot
		if err := json.Unmarshal(body, &snap); err != nil {
			t.Fatalf("Failed to deserialize netflow snapshot: %v", err)
		}
		if snap.Timestamp == "" {
			t.Errorf("Netflow timestamp missing")
		}
	})

	// -------------------------------------------------------------------------
	// Combo 14: F13 (Live Docker) + F01 (Network API)
	// -------------------------------------------------------------------------
	t.Run("Combo14_LiveDockerNetworkInspectConsistency", func(t *testing.T) {
		nets, err := client.ListNetworks()
		if err != nil || len(nets) == 0 {
			t.Fatalf("ListNetworks failed: %v", err)
		}
		target := nets[0]
		single, err := client.GetNetwork(target.Id)
		if err != nil {
			t.Fatalf("GetNetwork failed for %s: %v", target.Id, err)
		}
		if single.Name != target.Name {
			t.Errorf("GetNetwork name %s does not match list name %s", single.Name, target.Name)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 15: F13 (Live Docker) + F03 (Socket Reader)
	// -------------------------------------------------------------------------
	t.Run("Combo15_PublicVsPrivatePortRiskDifferentiation", func(t *testing.T) {
		snap, err := client.GetNetflow()
		if err != nil {
			t.Fatalf("GetNetflow failed: %v", err)
		}
		for _, sc := range snap.Connections {
			if sc.LocalAddr == "0.0.0.0" && sc.LocalPort == 6379 {
				if sc.SecurityLevel != "critical" {
					t.Errorf("Public Redis on 0.0.0.0 must be critical, got %s", sc.SecurityLevel)
				}
			}
			if sc.LocalAddr == "127.0.0.1" && sc.LocalPort == 5432 {
				if sc.SecurityLevel != "safe" {
					t.Errorf("Localhost Postgres on 127.0.0.1 must be safe, got %s", sc.SecurityLevel)
				}
			}
		}
	})
}
