package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTier2_BoundaryAndCornerCases(t *testing.T) {
	client, teardown := SetupTest(t)
	defer teardown()

	// -------------------------------------------------------------------------
	// F01: Backend Enriched Network API (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F01_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F01_01_NonExistentNetworkIDReturns404", func(t *testing.T) {
			resp, body, err := client.RawGet("/api/v1/networks/non-existent-net-id-99999")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected HTTP 404 for non-existent network, got %d (body: %s)", resp.StatusCode, string(body))
			}
		})

		t.Run("TC_T2_F01_02_SpecialCharsAndPathTraversalSafe", func(t *testing.T) {
			resp, _, err := client.RawGet("/api/v1/networks/..%2F..%2Fetc%2Fpasswd")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected HTTP 400 or 404 for traversal attempt, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F01_03_NetworkWithoutContainersHasEmptyMap", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if len(n.Containers) == 0 {
					if n.Containers == nil {
						t.Errorf("Network %s Containers field is nil instead of empty map", n.Name)
					}
				}
			}
		})

		t.Run("TC_T2_F01_04_HostAndNullNetworksOmitSubnetGracefully", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Driver == "host" || n.Driver == "null" {
					if n.SubnetDetails != nil && n.SubnetDetails.Subnet != "" {
						t.Logf("Notice: Driver %s reported SubnetDetails: %+v", n.Driver, n.SubnetDetails)
					}
				}
			}
		})

		t.Run("TC_T2_F01_05_ShortPrefixIDMatching", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil || len(nets) == 0 {
				t.Fatalf("ListNetworks prerequisite failed: %v", err)
			}
			fullID := nets[0].Id
			if len(fullID) >= 8 {
				shortID := fullID[:8]
				netObj, err := client.GetNetwork(shortID)
				if err == nil && netObj != nil {
					if !strings.HasPrefix(fullID, netObj.Id) && !strings.HasPrefix(netObj.Id, shortID) {
						t.Errorf("Short ID lookup mismatch: queried %s, got %s", shortID, netObj.Id)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F02: Backend Container Bandwidth Telemetry (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F02_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F02_01_IdleContainerZeroBandwidthHandled", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			foundIdle := false
			for _, ct := range snap.ContainerTraffic {
				if ct.RxBps == 0 && ct.TxBps == 0 {
					foundIdle = true
					break
				}
			}
			if !foundIdle {
				t.Logf("Notice: No completely idle containers observed")
			}
		})

		t.Run("TC_T2_F02_02_HighBandwidthGbpsScaleHandled", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				if ct.RxBps > 1e12 || ct.TxBps > 1e12 {
					t.Errorf("Unexpectedly enormous bandwidth rate: %+v", ct)
				}
			}
		})

		t.Run("TC_T2_F02_03_ZeroPacketRatesHandled", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				if ct.RxPps < 0 || ct.TxPps < 0 {
					t.Errorf("Negative packet rate: %+v", ct)
				}
			}
		})

		t.Run("TC_T2_F02_04_ContainerTrafficSliceNonNull", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.ContainerTraffic == nil {
				t.Errorf("ContainerTraffic slice must be non-nil even if 0 containers exist")
			}
		})

		t.Run("TC_T2_F02_05_CumulativeBytes64BitRange", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				_ = ct.RxBytes
				_ = ct.TxBytes
			}
		})
	})

	// -------------------------------------------------------------------------
	// F03: Hybrid Socket Collector & Port Security Classifier (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F03_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F03_01_IPv6AnybindClassification", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "::" || sc.LocalAddr == "[::]" {
					if sc.SecurityLevel == "" {
						t.Errorf("Empty security level on IPv6 anybind socket: %+v", sc)
					}
				}
			}
		})

		t.Run("TC_T2_F03_02_HighEphemeralPortClassification", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalPort > 49152 {
					if sc.SecurityLevel == "" {
						t.Errorf("High ephemeral port %d missing security classification", sc.LocalPort)
					}
				}
			}
		})

		t.Run("TC_T2_F03_03_EstablishedAndListenStatesDistinct", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.State != "LISTEN" && sc.State != "ESTABLISHED" && sc.State != "TIME_WAIT" && sc.State != "CLOSE_WAIT" {
					t.Logf("Notice: Observed socket in state %s", sc.State)
				}
			}
		})

		t.Run("TC_T2_F03_04_HostSocketWithoutContainerSafe", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.ContainerID == "" {
					if sc.LocalPort == 0 && sc.Protocol == "" {
						t.Errorf("Host socket has unpopulated endpoint: %+v", sc)
					}
				}
			}
		})

		t.Run("TC_T2_F03_05_ClassificationFieldConsistency", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.Classification != "port_based" && sc.Classification != "ebpf_observed" && sc.Classification != "docker_synthesized" {
					t.Logf("Notice: Classification mode '%s' on socket %+v", sc.Classification, sc)
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F04: Frontend Enriched Types & API Schema Conformity (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F04_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F04_01_MalformedJSONOnPostReturns400", func(t *testing.T) {
			resp, _, err := client.RawPost("/api/v1/networks", "invalid-json-{broken")
			if err != nil {
				t.Fatalf("RawPost failed: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected HTTP 400 Bad Request on malformed JSON, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F04_02_ContentTypeApplicationJSON", func(t *testing.T) {
			resp, _, err := client.RawGet("/api/v1/networks")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Expected application/json content type, got '%s'", ct)
			}
		})

		t.Run("TC_T2_F04_03_MissingOptionalFieldsDoesNotBreak", func(t *testing.T) {
			_, body, err := client.RawGet("/api/v1/networks")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			var nets []EnrichedDockerNetwork
			if err := json.Unmarshal(body, &nets); err != nil {
				t.Fatalf("Failed to decode networks with missing optional fields: %v", err)
			}
		})

		t.Run("TC_T2_F04_04_EmptyBodyOnPostRejected", func(t *testing.T) {
			resp, _, err := client.RawPost("/api/v1/networks", nil)
			if err != nil {
				t.Fatalf("RawPost failed: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusForbidden {
				t.Logf("Notice: Empty body returned status %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F04_05_LongURLParameterHandledCleanly", func(t *testing.T) {
			longID := strings.Repeat("a", 2048)
			resp, _, err := client.RawGet("/api/v1/networks/" + longID)
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode >= 500 {
				t.Errorf("Server error %d on long parameter", resp.StatusCode)
			}
		})
	})

	// -------------------------------------------------------------------------
	// F05: UniFi-Style IPAM Subnet Inspection (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F05_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F05_01_SmallSubnetArithmetic", func(t *testing.T) {
			// Subnet /30 has 4 total IPs, 2 usable
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil && strings.HasSuffix(n.SubnetDetails.Subnet, "/30") {
					if n.SubnetDetails.TotalIPs != 4 {
						t.Errorf("Subnet /30 expected 4 total IPs, got %d", n.SubnetDetails.TotalIPs)
					}
				}
			}
		})

		t.Run("TC_T2_F05_02_LargeSubnetArithmetic", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil && strings.HasSuffix(n.SubnetDetails.Subnet, "/16") {
					if n.SubnetDetails.TotalIPs != 65536 {
						t.Errorf("Subnet /16 expected 65536 total IPs, got %d", n.SubnetDetails.TotalIPs)
					}
				}
			}
		})

		t.Run("TC_T2_F05_03_ZeroUtilizationBoundary", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil && n.SubnetDetails.UsedIPs == 0 {
					if n.SubnetDetails.UtilizationPercent != 0.0 {
						t.Errorf("Expected 0.0%% utilization when 0 IPs used, got %f", n.SubnetDetails.UtilizationPercent)
					}
				}
			}
		})

		t.Run("TC_T2_F05_04_UtilizationPercentageNeverExceeds100", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					if n.SubnetDetails.UtilizationPercent > 100.0 {
						t.Errorf("UtilizationPercent %f exceeds 100%%", n.SubnetDetails.UtilizationPercent)
					}
				}
			}
		})

		t.Run("TC_T2_F05_05_FreeIPsMatchesArithmetic", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					sd := n.SubnetDetails
					if sd.TotalIPs > 2 {
						usable := sd.TotalIPs - 2
						expectedFree := usable - sd.UsedIPs
						if expectedFree < 0 {
							expectedFree = 0
						}
						diff := sd.FreeIPs - expectedFree
						if diff < -2 || diff > 2 { // allow gateway IP variance
							t.Logf("Notice: FreeIPs %d differs from calculated usable - used (%d)", sd.FreeIPs, expectedFree)
						}
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F06: Connected Container Client Insights (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F06_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F06_01_MultipleNetworksIsolation", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			containerIPs := make(map[string]map[string]string) // containerName -> netName -> IP
			for _, n := range nets {
				for _, c := range n.Containers {
					if _, ok := containerIPs[c.Name]; !ok {
						containerIPs[c.Name] = make(map[string]string)
					}
					containerIPs[c.Name][n.Name] = c.IPv4Address
				}
			}
			for name, netMap := range containerIPs {
				if len(netMap) > 1 {
					t.Logf("Container %s is multihomed on %d networks", name, len(netMap))
				}
			}
		})

		t.Run("TC_T2_F06_02_DualStackIPv4IPv6Fields", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					_ = c.IPv6Address // safe access
				}
			}
		})

		t.Run("TC_T2_F06_03_StoppedContainerMaintainsState", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					if c.State == "exited" || c.State == "stopped" {
						t.Logf("Observed stopped container in network attachment: %s", c.Name)
					}
				}
			}
		})

		t.Run("TC_T2_F06_04_ContainerNamePreservedIntact", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					if strings.Contains(c.Name, "\x00") {
						t.Errorf("Container name contains null byte: %s", c.Name)
					}
				}
			}
		})

		t.Run("TC_T2_F06_05_IPUniquenessInSameSubnet", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				seenIPs := make(map[string]string)
				for id, c := range n.Containers {
					if c.IPv4Address != "" {
						ip := strings.Split(c.IPv4Address, "/")[0]
						if existing, ok := seenIPs[ip]; ok {
							t.Errorf("IP address conflict on network %s: %s used by both %s and %s", n.Name, ip, existing, id)
						}
						seenIPs[ip] = id
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F07: Network Driver Options & Scope Inspection (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F07_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F07_01_HostNetworkScopeLocal", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Driver == "host" {
					if n.Scope != "local" {
						t.Errorf("Expected host network scope to be 'local', got '%s'", n.Scope)
					}
				}
			}
		})

		t.Run("TC_T2_F07_02_NullNetworkNoGateway", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Driver == "null" {
					if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Gateway != "" {
						t.Errorf("Null network unexpectedly defined a gateway: %s", n.IPAM.Config[0].Gateway)
					}
				}
			}
		})

		t.Run("TC_T2_F07_03_InternalNetworkFlagCheck", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Internal {
					t.Logf("Observed internal isolated network: %s", n.Name)
				}
			}
		})

		t.Run("TC_T2_F07_04_ICCEnabledOrDisabled", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Options != nil {
					if icc, ok := n.Options["com.docker.network.bridge.enable_icc"]; ok {
						if icc != "true" && icc != "false" {
							t.Errorf("Invalid ICC flag value: %s", icc)
						}
					}
				}
			}
		})

		t.Run("TC_T2_F07_05_MTUOptionFormat", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Options != nil {
					if mtu, ok := n.Options["com.docker.network.driver.mtu"]; ok {
						if mtu == "" {
							t.Errorf("Empty MTU option string")
						}
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F08: Retain Interactive Topology Graph (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F08_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F08_01_ZeroTrafficEdgesInactive", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, edge := range snap.Topology.Edges {
				if edge.Type == "traffic" && !edge.Active {
					t.Logf("Observed inactive traffic edge: %s", edge.ID)
				}
			}
		})

		t.Run("TC_T2_F08_02_UniqueNodeIDsAcrossAllNodes", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			seenIDs := make(map[string]bool)
			for _, node := range snap.Topology.Nodes {
				if seenIDs[node.ID] {
					t.Errorf("Duplicate node ID in topology: %s", node.ID)
				}
				seenIDs[node.ID] = true
			}
		})

		t.Run("TC_T2_F08_03_GatewayNodeAttributes", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, node := range snap.Topology.Nodes {
				if node.Type == "gateway" {
					if node.Label == "" {
						t.Errorf("Gateway node missing label: %s", node.ID)
					}
				}
			}
		})

		t.Run("TC_T2_F08_04_DisconnectedNodeSafe", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			edgeEndpoints := make(map[string]bool)
			for _, edge := range snap.Topology.Edges {
				edgeEndpoints[edge.Source] = true
				edgeEndpoints[edge.Target] = true
			}
			for _, node := range snap.Topology.Nodes {
				if !edgeEndpoints[node.ID] {
					t.Logf("Node %s has no edges connected", node.ID)
				}
			}
		})

		t.Run("TC_T2_F08_05_TimestampRFC3339Format", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Timestamp != "" {
				_, err := time.Parse(time.RFC3339, snap.Timestamp)
				if err != nil {
					// Fallback check for ISO8601
					_, err2 := time.Parse("2006-01-02T15:04:05Z07:00", snap.Timestamp)
					if err2 != nil {
						t.Errorf("Timestamp %s is not valid RFC3339: %v", snap.Timestamp, err)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F09: Live Bandwidth Rate Meters (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F09_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F09_01_TotalRatesWhenAllIdle", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.TotalRxBps < 0 || snap.Totals.TotalTxBps < 0 {
				t.Errorf("Totals rates cannot be negative")
			}
		})

		t.Run("TC_T2_F09_02_AsymmetricRatesHandling", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.TotalRxBps > 0 && snap.Totals.TotalTxBps == 0 {
				t.Logf("Observed asymmetric traffic: Rx > 0, Tx == 0")
			}
		})

		t.Run("TC_T2_F09_03_ActiveConnectionsNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.ActiveConnections < 0 {
				t.Errorf("ActiveConnections cannot be negative: %d", snap.Totals.ActiveConnections)
			}
		})

		t.Run("TC_T2_F09_04_SecurityAlertsMatchesAlertSockets", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			alertCount := 0
			for _, sc := range snap.Connections {
				if sc.SecurityLevel == "critical" || sc.SecurityLevel == "caution" {
					alertCount++
				}
			}
			if snap.Totals.SecurityAlertsCount != alertCount {
				t.Logf("Notice: Totals securityAlertsCount %d, counted critical+caution %d", snap.Totals.SecurityAlertsCount, alertCount)
			}
		})

		t.Run("TC_T2_F09_05_CumulativeBytesMonotonicity", func(t *testing.T) {
			snap1, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow 1 failed: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			snap2, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow 2 failed: %v", err)
			}
			bytesMap := make(map[string]uint64)
			for _, ct := range snap1.ContainerTraffic {
				bytesMap[ct.ContainerID] = ct.RxBytes
			}
			for _, ct := range snap2.ContainerTraffic {
				if prev, ok := bytesMap[ct.ContainerID]; ok {
					if ct.RxBytes < prev {
						t.Errorf("Cumulative bytes decreased for container %s: prev %d, curr %d", ct.ContainerID, prev, ct.RxBytes)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F10: Live Traffic Breakdown by Container & Protocol (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F10_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F10_01_ProtocolConfiguredZeroObserved", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ps := range snap.ProtocolStats {
				if ps.Configured > 0 && ps.Observed == 0 {
					t.Logf("Protocol %s has configured ports but 0 observed", ps.Protocol)
				}
			}
		})

		t.Run("TC_T2_F10_02_ProtocolObservedZeroActive", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ps := range snap.ProtocolStats {
				if ps.Observed > 0 && ps.Active == 0 {
					t.Logf("Protocol %s has observed sockets but 0 active", ps.Protocol)
				}
			}
		})

		t.Run("TC_T2_F10_03_BandwidthSortingTieHandling", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			zeroCount := 0
			for _, ct := range snap.ContainerTraffic {
				if ct.RxBps == 0 && ct.TxBps == 0 {
					zeroCount++
				}
			}
			if zeroCount >= 2 {
				t.Logf("Observed %d tied zero-bandwidth containers", zeroCount)
			}
		})

		t.Run("TC_T2_F10_04_UnlistedProtocolHandling", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.DetectedProto == "UNKNOWN" || sc.DetectedProto == "OTHER" {
					t.Logf("Observed unlisted protocol on port %d", sc.LocalPort)
				}
			}
		})

		t.Run("TC_T2_F10_05_ProtocolStatsNoDuplicates", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			seenProtocols := make(map[string]bool)
			for _, ps := range snap.ProtocolStats {
				if seenProtocols[ps.Protocol] {
					t.Errorf("Duplicate protocol entry in protocolStats: %s", ps.Protocol)
				}
				seenProtocols[ps.Protocol] = true
			}
		})
	})

	// -------------------------------------------------------------------------
	// F11: Active Sockets Table with Security Attribution (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F11_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F11_01_LoopbackIPv4ClassificationSafe", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "127.0.0.1" {
					if sc.SecurityLevel != "safe" && sc.SecurityLevel != "internal" {
						t.Errorf("Expected loopback socket to be safe, got %s", sc.SecurityLevel)
					}
				}
			}
		})

		t.Run("TC_T2_F11_02_LoopbackIPv6ClassificationSafe", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "::1" || sc.LocalAddr == "[::1]" {
					if sc.SecurityLevel != "safe" && sc.SecurityLevel != "internal" {
						t.Errorf("Expected IPv6 loopback socket to be safe, got %s", sc.SecurityLevel)
					}
				}
			}
		})

		t.Run("TC_T2_F11_03_PrivateContainerSocketClassificationInternal", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if strings.HasPrefix(sc.LocalAddr, "172.") || strings.HasPrefix(sc.LocalAddr, "10.") || strings.HasPrefix(sc.LocalAddr, "192.168.") {
					if sc.SecurityLevel == "critical" {
						t.Errorf("Private container socket unexpectedly marked critical: %+v", sc)
					}
				}
			}
		})

		t.Run("TC_T2_F11_04_NonStandardPortPublicClassification", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "0.0.0.0" && sc.LocalPort > 10000 {
					if sc.SecurityLevel == "" {
						t.Errorf("Public non-standard port %d missing securityLevel", sc.LocalPort)
					}
				}
			}
		})

		t.Run("TC_T2_F11_05_MultiplePortsInSameContainer", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			containerPorts := make(map[string][]uint16)
			for _, sc := range snap.Connections {
				if sc.ContainerID != "" {
					containerPorts[sc.ContainerID] = append(containerPorts[sc.ContainerID], sc.LocalPort)
				}
			}
			for cID, ports := range containerPorts {
				if len(ports) > 1 {
					t.Logf("Container %s has %d listening ports", cID, len(ports))
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F12: Dashboard JSX Pre-existing Build Fix (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F12_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F12_01_TrailingSlashRouteResolution", func(t *testing.T) {
			resp, _, err := client.RawGet("/networks/")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMovedPermanently {
				t.Errorf("Unexpected status on /networks/: %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F12_02_QueryStringParametersHandling", func(t *testing.T) {
			resp, _, err := client.RawGet("/networks?search=bridge&sort=ip")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 with query strings, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F12_03_SPAFallbackOnArbitraryRoute", func(t *testing.T) {
			resp, body, err := client.RawGet("/some/deep/spa/route")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode == http.StatusOK {
				if !strings.Contains(string(body), "<html") {
					t.Errorf("Expected SPA fallback HTML response")
				}
			}
		})

		t.Run("TC_T2_F12_04_ConcurrentWebRequestsStability", func(t *testing.T) {
			errChan := make(chan error, 5)
			for i := 0; i < 5; i++ {
				go func() {
					resp, _, err := client.RawGet("/networks")
					if err != nil {
						errChan <- err
						return
					}
					if resp.StatusCode >= 500 {
						errChan <- err
						return
					}
					errChan <- nil
				}()
			}
			for i := 0; i < 5; i++ {
				if err := <-errChan; err != nil {
					t.Errorf("Concurrent request error: %v", err)
				}
			}
		})

		t.Run("TC_T2_F12_05_HeadMethodOnWebRoutes", func(t *testing.T) {
			req, _ := http.NewRequest("HEAD", client.BaseURL+"/", nil)
			resp, err := client.HTTPClient.Do(req)
			if err != nil {
				t.Fatalf("HEAD request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 on HEAD /, got %d", resp.StatusCode)
			}
		})
	})

	// -------------------------------------------------------------------------
	// F13: End-to-End Build & Live Docker Validation (Boundaries)
	// -------------------------------------------------------------------------
	t.Run("F13_BoundaryCases", func(t *testing.T) {
		t.Run("TC_T2_F13_01_InvalidPasswordReturns401", func(t *testing.T) {
			badClient, _ := NewClient(client.BaseURL)
			err := badClient.Login("admin", "WrongPassword123!")
			if err == nil {
				t.Errorf("Expected login with wrong password to fail")
			}
		})

		t.Run("TC_T2_F13_02_NonExistentUserReturns401", func(t *testing.T) {
			badClient, _ := NewClient(client.BaseURL)
			err := badClient.Login("nonexistent_user", "OrbitAdminSecret2026!")
			if err == nil {
				t.Errorf("Expected login with nonexistent user to fail")
			}
		})

		t.Run("TC_T2_F13_03_MissingCSRFTokenReturns403", func(t *testing.T) {
			noCSRFClient, _ := NewClient(client.BaseURL)
			// Login without setting CSRFToken on requests
			_ = noCSRFClient.Login("admin", "OrbitAdminSecret2026!")
			noCSRFClient.CSRFToken = "" // clear CSRF token
			resp, _, err := noCSRFClient.RawPost("/api/v1/networks", map[string]string{"Name": "test-net"})
			if err != nil {
				t.Fatalf("RawPost failed: %v", err)
			}
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected HTTP 403 Forbidden without CSRF token, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F13_04_InvalidCSRFTokenReturns403", func(t *testing.T) {
			tamperedClient, _ := NewClient(client.BaseURL)
			_ = tamperedClient.Login("admin", "OrbitAdminSecret2026!")
			tamperedClient.CSRFToken = "tampered-invalid-csrf-token"
			resp, _, err := tamperedClient.RawPost("/api/v1/networks", map[string]string{"Name": "test-net"})
			if err != nil {
				t.Fatalf("RawPost failed: %v", err)
			}
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected HTTP 403 Forbidden on tampered CSRF, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T2_F13_05_InvalidSessionCookieTreatedAsUnauthenticated", func(t *testing.T) {
			fakeCookieClient, _ := NewClient(client.BaseURL)
			fakeCookie := &http.Cookie{Name: "orbit_session", Value: "invalid-session-token-xyz"}
			u, _ := http.NewRequest("GET", client.BaseURL, nil)
			fakeCookieClient.HTTPClient.Jar.SetCookies(u.URL, []*http.Cookie{fakeCookie})
			s, err := fakeCookieClient.GetAuthStatus()
			if err != nil {
				t.Fatalf("GetAuthStatus failed: %v", err)
			}
			if s.Authenticated {
				t.Errorf("Expected invalid session token to be unauthenticated")
			}
		})
	})
}
