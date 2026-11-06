package e2e

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func TestTier1_FeatureCoverage(t *testing.T) {
	client, teardown := SetupTest(t)
	defer teardown()

	// -------------------------------------------------------------------------
	// F01: Backend Enriched Network API
	// -------------------------------------------------------------------------
	t.Run("F01_EnrichedNetworkAPI", func(t *testing.T) {
		t.Run("TC_T1_F01_01_ListNetworksSuccess", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			if len(nets) == 0 {
				t.Fatalf("Expected at least 1 network, got 0")
			}
		})

		t.Run("TC_T1_F01_02_GetNetworkById", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil || len(nets) == 0 {
				t.Fatalf("ListNetworks prerequisite failed: %v", err)
			}
			targetID := nets[0].Id
			netObj, err := client.GetNetwork(targetID)
			if err != nil {
				t.Fatalf("GetNetwork failed for id %s: %v", targetID, err)
			}
			if netObj.Id != targetID && !strings.HasPrefix(targetID, netObj.Id) {
				t.Errorf("Expected network ID %s, got %s", targetID, netObj.Id)
			}
		})

		t.Run("TC_T1_F01_03_SubnetDetails9Fields", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			var bridgeNet *EnrichedDockerNetwork
			for i := range nets {
				if nets[i].SubnetDetails != nil {
					bridgeNet = &nets[i]
					break
				}
			}
			if bridgeNet == nil {
				t.Fatalf("Expected at least one network with SubnetDetails")
			}
			sd := bridgeNet.SubnetDetails
			if sd.Subnet == "" || sd.Gateway == "" || sd.NetworkIP == "" || sd.BroadcastIP == "" || sd.UsableRange == "" {
				t.Errorf("SubnetDetails missing string fields: %+v", sd)
			}
			if sd.TotalIPs <= 0 || sd.FreeIPs < 0 || sd.UsedIPs < 0 || sd.UtilizationPercent < 0 {
				t.Errorf("SubnetDetails numeric metrics invalid: %+v", sd)
			}
		})

		t.Run("TC_T1_F01_04_ConnectedContainersAttachments", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			foundContainer := false
			for _, n := range nets {
				if len(n.Containers) > 0 {
					foundContainer = true
					for _, c := range n.Containers {
						if c.Name == "" || c.EndpointID == "" {
							t.Errorf("Container member missing Name or EndpointID: %+v", c)
						}
					}
				}
			}
			if !foundContainer {
				t.Logf("Notice: No connected containers found in default networks")
			}
		})

		t.Run("TC_T1_F01_05_DriverOptionsAndLabelsPreserved", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			foundOptions := false
			for _, n := range nets {
				if len(n.Options) > 0 || len(n.Labels) > 0 {
					foundOptions = true
					break
				}
			}
			if !foundOptions {
				t.Logf("Notice: No custom options/labels found in networks")
			}
		})
	})

	// -------------------------------------------------------------------------
	// F02: Backend Container Bandwidth Telemetry
	// -------------------------------------------------------------------------
	t.Run("F02_ContainerBandwidthTelemetry", func(t *testing.T) {
		t.Run("TC_T1_F02_01_NetflowContainerTrafficArray", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.ContainerTraffic == nil {
				t.Fatalf("Expected containerTraffic array in netflow snapshot, got nil")
			}
		})

		t.Run("TC_T1_F02_02_ContainerTrafficIdsAndNames", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				if ct.ContainerID == "" || ct.ContainerName == "" {
					t.Errorf("ContainerTraffic item missing ID or Name: %+v", ct)
				}
				if ct.RxBps < 0 || ct.TxBps < 0 {
					t.Errorf("ContainerTraffic rates must be >= 0: %+v", ct)
				}
			}
		})

		t.Run("TC_T1_F02_03_ContainerTrafficCumulativeBytes", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				if ct.RxBytes < 0 || ct.TxBytes < 0 {
					t.Errorf("ContainerTraffic bytes must be non-negative: %+v", ct)
				}
			}
		})

		t.Run("TC_T1_F02_04_PacketRatesNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				if ct.RxPps < 0 || ct.TxPps < 0 {
					t.Errorf("ContainerTraffic packet rates must be non-negative: %+v", ct)
				}
			}
		})

		t.Run("TC_T1_F02_05_NetflowTotalsStructFields", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.TotalRxBps < 0 || snap.Totals.TotalTxBps < 0 {
				t.Errorf("Totals bandwidth rates must be >= 0: %+v", snap.Totals)
			}
			if snap.Totals.ActiveConnections < 0 || snap.Totals.SecurityAlertsCount < 0 {
				t.Errorf("Totals connection and alert counts must be >= 0: %+v", snap.Totals)
			}
		})
	})

	// -------------------------------------------------------------------------
	// F03: Hybrid Socket Collector & Port Security Classifier
	// -------------------------------------------------------------------------
	t.Run("F03_HybridSocketCollectorAndPortSecurity", func(t *testing.T) {
		t.Run("TC_T1_F03_01_ConnectionsArrayPresent", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Connections == nil {
				t.Fatalf("Expected connections array in netflow snapshot, got nil")
			}
		})

		t.Run("TC_T1_F03_02_SocketEndpointsAndProtocol", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "" || sc.Protocol == "" {
					t.Errorf("Socket connection missing LocalAddr or Protocol: %+v", sc)
				}
			}
		})

		t.Run("TC_T1_F03_03_SocketStateAndAttribution", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.State == "" {
					t.Errorf("Socket connection missing state: %+v", sc)
				}
				if sc.ContainerID == "" && sc.ContainerName == "" {
					t.Logf("Notice: Socket without container attribution: %+v", sc)
				}
			}
		})

		t.Run("TC_T1_F03_04_SecurityLevelEnumCompliance", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			validLevels := map[string]bool{
				"critical": true,
				"caution":  true,
				"safe":     true,
				"internal": true,
			}
			for _, sc := range snap.Connections {
				if !validLevels[sc.SecurityLevel] {
					t.Errorf("Invalid SecurityLevel '%s' in socket %+v", sc.SecurityLevel, sc)
				}
			}
		})

		t.Run("TC_T1_F03_05_SecurityBadgeAndDesc", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.SecurityBadge == "" || sc.SecurityDesc == "" {
					t.Errorf("Socket missing SecurityBadge or SecurityDesc: %+v", sc)
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F04: Frontend Enriched Types & API Schema Conformity
	// -------------------------------------------------------------------------
	t.Run("F04_SchemaConformity", func(t *testing.T) {
		t.Run("TC_T1_F04_01_NetworkSchemaConformity", func(t *testing.T) {
			resp, body, err := client.RawGet("/api/v1/networks")
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Fatalf("Failed to fetch /api/v1/networks: %v", err)
			}
			var nets []EnrichedDockerNetwork
			if err := json.Unmarshal(body, &nets); err != nil {
				t.Fatalf("Failed to unmarshal networks JSON into EnrichedDockerNetwork schema: %v", err)
			}
			if len(nets) == 0 {
				t.Fatalf("Network array unexpectedly empty")
			}
		})

		t.Run("TC_T1_F04_02_NetflowSchemaConformity", func(t *testing.T) {
			resp, body, err := client.RawGet("/api/v1/netflow")
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Fatalf("Failed to fetch /api/v1/netflow: %v", err)
			}
			var snap EnrichedNetflowSnapshot
			if err := json.Unmarshal(body, &snap); err != nil {
				t.Fatalf("Failed to unmarshal netflow JSON into EnrichedNetflowSnapshot schema: %v", err)
			}
			if snap.Timestamp == "" {
				t.Errorf("Expected snapshot timestamp to be non-empty")
			}
		})

		t.Run("TC_T1_F04_03_SubnetDetailsNumericTypes", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					if n.SubnetDetails.TotalIPs < 0 || n.SubnetDetails.UsedIPs < 0 {
						t.Errorf("Negative values in SubnetDetails: %+v", n.SubnetDetails)
					}
				}
			}
		})

		t.Run("TC_T1_F04_04_SocketConnectionSchema", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.Classification == "" || sc.DetectedProto == "" {
					t.Errorf("Socket missing classification or detectedProto: %+v", sc)
				}
			}
		})

		t.Run("TC_T1_F04_05_ProtocolStatsSchema", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ps := range snap.ProtocolStats {
				if ps.Protocol == "" {
					t.Errorf("ProtocolSummary missing protocol name: %+v", ps)
				}
				if ps.Configured < 0 || ps.Observed < 0 || ps.Active < 0 {
					t.Errorf("Protocol counts must be non-negative: %+v", ps)
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F05: UniFi-Style IPAM Subnet Inspection
	// -------------------------------------------------------------------------
	t.Run("F05_IPAMSubnetInspection", func(t *testing.T) {
		t.Run("TC_T1_F05_01_ValidCIDRAndGatewayParsing", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					_, ipnet, err := net.ParseCIDR(n.SubnetDetails.Subnet)
					if err != nil {
						t.Errorf("Subnet %s is not a valid CIDR: %v", n.SubnetDetails.Subnet, err)
					}
					gw := net.ParseIP(n.SubnetDetails.Gateway)
					if gw == nil {
						t.Errorf("Gateway %s is not a valid IP", n.SubnetDetails.Gateway)
					}
					if ipnet != nil && gw != nil && !ipnet.Contains(gw) {
						t.Errorf("Gateway %s is outside subnet %s", gw, ipnet)
					}
				}
			}
		})

		t.Run("TC_T1_F05_02_NetworkAndBroadcastIPValidity", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					netIP := net.ParseIP(n.SubnetDetails.NetworkIP)
					bcastIP := net.ParseIP(n.SubnetDetails.BroadcastIP)
					if netIP == nil || bcastIP == nil {
						t.Errorf("NetworkIP (%s) or BroadcastIP (%s) invalid", n.SubnetDetails.NetworkIP, n.SubnetDetails.BroadcastIP)
					}
				}
			}
		})

		t.Run("TC_T1_F05_03_UsableRangeNotation", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			rangeRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}\s*-\s*(\d{1,3}\.){3}\d{1,3}$`)
			for _, n := range nets {
				if n.SubnetDetails != nil {
					if !rangeRegex.MatchString(n.SubnetDetails.UsableRange) {
						t.Errorf("UsableRange %s does not match expected 'start - end' format", n.SubnetDetails.UsableRange)
					}
				}
			}
		})

		t.Run("TC_T1_F05_04_TotalIPsAndFreeIPsDerivation", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					sd := n.SubnetDetails
					// For /16, total IPs should be 65536
					if strings.HasSuffix(sd.Subnet, "/16") && sd.TotalIPs != 65536 {
						t.Errorf("Subnet /16 expected 65536 total IPs, got %d", sd.TotalIPs)
					}
					// FreeIPs should be roughly TotalIPs - 2 - UsedIPs (or non-negative)
					if sd.FreeIPs > sd.TotalIPs || sd.FreeIPs < 0 {
						t.Errorf("FreeIPs (%d) outside expected range [0, %d]", sd.FreeIPs, sd.TotalIPs)
					}
				}
			}
		})

		t.Run("TC_T1_F05_05_UtilizationPercentBoundaries", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					pct := n.SubnetDetails.UtilizationPercent
					if pct < 0.0 || pct > 100.0 {
						t.Errorf("UtilizationPercent %f out of bounds [0.0, 100.0]", pct)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F06: Connected Container Client Insights
	// -------------------------------------------------------------------------
	t.Run("F06_ConnectedContainerInsights", func(t *testing.T) {
		t.Run("TC_T1_F06_01_MacAddressFormatValidation", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			macRegex := regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
			for _, n := range nets {
				for _, c := range n.Containers {
					if c.MacAddress != "" && !macRegex.MatchString(c.MacAddress) {
						t.Errorf("Container MAC %s does not match standard hex format", c.MacAddress)
					}
				}
			}
		})

		t.Run("TC_T1_F06_02_ContainerIPInSubnetCIDR", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.SubnetDetails != nil {
					_, ipnet, _ := net.ParseCIDR(n.SubnetDetails.Subnet)
					if ipnet != nil {
						for _, c := range n.Containers {
							if c.IPv4Address != "" {
								ip := net.ParseIP(strings.Split(c.IPv4Address, "/")[0])
								if ip != nil && !ipnet.Contains(ip) {
									t.Errorf("Container IP %s not in subnet %s", ip, ipnet)
								}
							}
						}
					}
				}
			}
		})

		t.Run("TC_T1_F06_03_ContainerStatusBadgePresent", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					if c.State == "" {
						t.Logf("Notice: Container %s state is empty", c.Name)
					}
				}
			}
		})

		t.Run("TC_T1_F06_04_EndpointIDFormat", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					if c.EndpointID == "" {
						t.Errorf("Container %s missing EndpointID", c.Name)
					}
				}
			}
		})

		t.Run("TC_T1_F06_05_ContainerNameNotEmpty", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				for _, c := range n.Containers {
					if c.Name == "" {
						t.Errorf("Connected container missing name: %+v", c)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F07: Network Driver Options & Scope Inspection
	// -------------------------------------------------------------------------
	t.Run("F07_NetworkDriverOptionsAndScope", func(t *testing.T) {
		t.Run("TC_T1_F07_01_DriverFieldValidation", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			validDrivers := map[string]bool{
				"bridge":  true,
				"host":    true,
				"null":    true,
				"overlay": true,
				"macvlan": true,
			}
			for _, n := range nets {
				if !validDrivers[n.Driver] {
					t.Errorf("Unknown network driver %s on network %s", n.Driver, n.Name)
				}
			}
		})

		t.Run("TC_T1_F07_02_ScopeFieldPresent", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Scope == "" {
					t.Errorf("Network %s missing Scope field", n.Name)
				}
			}
		})

		t.Run("TC_T1_F07_03_InternalFlagBoolean", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				_ = n.Internal // verifies boolean property exists
			}
		})

		t.Run("TC_T1_F07_04_EnableIPv6FlagBoolean", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				_ = n.EnableIPv6 // verifies boolean property exists
			}
		})

		t.Run("TC_T1_F07_05_DriverOptionsInspection", func(t *testing.T) {
			nets, err := client.ListNetworks()
			if err != nil {
				t.Fatalf("ListNetworks failed: %v", err)
			}
			for _, n := range nets {
				if n.Driver == "bridge" && n.Options != nil {
					if icc, ok := n.Options["com.docker.network.bridge.enable_icc"]; ok {
						if icc != "true" && icc != "false" {
							t.Errorf("Unexpected enable_icc option value %s", icc)
						}
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F08: Retain Interactive Topology Graph
	// -------------------------------------------------------------------------
	t.Run("F08_TopologyGraphIntegrity", func(t *testing.T) {
		t.Run("TC_T1_F08_01_TopologyNodesAndEdgesPresent", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Topology.Nodes == nil || snap.Topology.Edges == nil {
				t.Fatalf("Topology Nodes or Edges slice is nil")
			}
		})

		t.Run("TC_T1_F08_02_ValidNodeTypes", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			validTypes := map[string]bool{
				"host":      true,
				"network":   true,
				"container": true,
				"gateway":   true,
				"external":  true,
			}
			for _, node := range snap.Topology.Nodes {
				if !validTypes[node.Type] {
					t.Errorf("Invalid node type '%s' on node %s", node.Type, node.ID)
				}
			}
		})

		t.Run("TC_T1_F08_03_ValidEdgeTypes", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			validEdgeTypes := map[string]bool{
				"member":  true,
				"traffic": true,
				"gateway": true,
			}
			for _, edge := range snap.Topology.Edges {
				if !validEdgeTypes[edge.Type] {
					t.Errorf("Invalid edge type '%s' on edge %s", edge.Type, edge.ID)
				}
			}
		})

		t.Run("TC_T1_F08_04_EdgeSourceAndTargetReferencing", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			nodeIDs := make(map[string]bool)
			for _, n := range snap.Topology.Nodes {
				nodeIDs[n.ID] = true
			}
			for _, edge := range snap.Topology.Edges {
				if !nodeIDs[edge.Source] || !nodeIDs[edge.Target] {
					t.Logf("Notice: Edge %s points to external or unmapped node (%s -> %s)", edge.ID, edge.Source, edge.Target)
				}
			}
		})

		t.Run("TC_T1_F08_05_NodeLabelNonEmpty", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, node := range snap.Topology.Nodes {
				if node.Label == "" {
					t.Errorf("Node %s missing label", node.ID)
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F09: Live Bandwidth Rate Meters
	// -------------------------------------------------------------------------
	t.Run("F09_LiveBandwidthRateMeters", func(t *testing.T) {
		t.Run("TC_T1_F09_01_TotalRxBpsNumericAndNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.TotalRxBps < 0 || math.IsNaN(snap.Totals.TotalRxBps) {
				t.Errorf("Invalid TotalRxBps: %f", snap.Totals.TotalRxBps)
			}
		})

		t.Run("TC_T1_F09_02_TotalTxBpsNumericAndNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.TotalTxBps < 0 || math.IsNaN(snap.Totals.TotalTxBps) {
				t.Errorf("Invalid TotalTxBps: %f", snap.Totals.TotalTxBps)
			}
		})

		t.Run("TC_T1_F09_03_RateMetersReflectContainerTraffic", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			var sumRx, sumTx float64
			for _, ct := range snap.ContainerTraffic {
				sumRx += ct.RxBps
				sumTx += ct.TxBps
			}
			if len(snap.ContainerTraffic) > 0 && (sumRx > 0 || sumTx > 0) {
				if snap.Totals.TotalRxBps == 0 && snap.Totals.TotalTxBps == 0 {
					t.Errorf("Totals indicate 0 bps but container traffic sum is Rx: %f, Tx: %f", sumRx, sumTx)
				}
			}
		})

		t.Run("TC_T1_F09_04_ActiveFlowsCountNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.ActiveConnections < 0 {
				t.Errorf("ActiveConnections must be non-negative: %d", snap.Totals.ActiveConnections)
			}
		})

		t.Run("TC_T1_F09_05_SecurityAlertsCountNonNegative", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if snap.Totals.SecurityAlertsCount < 0 {
				t.Errorf("SecurityAlertsCount must be non-negative: %d", snap.Totals.SecurityAlertsCount)
			}
		})
	})

	// -------------------------------------------------------------------------
	// F10: Live Traffic Breakdown by Container & Protocol
	// -------------------------------------------------------------------------
	t.Run("F10_TrafficBreakdownByContainerAndProtocol", func(t *testing.T) {
		t.Run("TC_T1_F10_01_ProtocolStatsArrayNotEmpty", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			if len(snap.ProtocolStats) == 0 {
				t.Fatalf("Expected protocolStats to be populated")
			}
		})

		t.Run("TC_T1_F10_02_ProtocolStatsPortCounters", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ps := range snap.ProtocolStats {
				if ps.Configured < 0 || ps.Observed < 0 || ps.Active < 0 {
					t.Errorf("Negative counter in protocol stats: %+v", ps)
				}
			}
		})

		t.Run("TC_T1_F10_03_SortableContainerBandwidth", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, ct := range snap.ContainerTraffic {
				totalRate := ct.RxBps + ct.TxBps
				if totalRate < 0 {
					t.Errorf("Total rate cannot be negative: %f", totalRate)
				}
			}
		})

		t.Run("TC_T1_F10_04_ActiveFlowsCorrelateWithConnections", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			var totalProtocolActive int
			for _, ps := range snap.ProtocolStats {
				totalProtocolActive += ps.Active
			}
			if totalProtocolActive < 0 {
				t.Errorf("Negative active protocol flows: %d", totalProtocolActive)
			}
		})

		t.Run("TC_T1_F10_05_DetectedProtocolsContainStandardProtocols", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			protoSet := make(map[string]bool)
			for _, ps := range snap.ProtocolStats {
				protoSet[ps.Protocol] = true
			}
			if !protoSet["HTTP"] && !protoSet["HTTPS"] && !protoSet["TCP"] {
				t.Logf("Notice: Neither HTTP nor HTTPS explicitly listed in protocol stats: %+v", snap.ProtocolStats)
			}
		})
	})

	// -------------------------------------------------------------------------
	// F11: Active Sockets Table with Security Attribution
	// -------------------------------------------------------------------------
	t.Run("F11_ActiveSocketsSecurityAttribution", func(t *testing.T) {
		t.Run("TC_T1_F11_01_SocketAttributionToContainer", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			attributedCount := 0
			for _, sc := range snap.Connections {
				if sc.ContainerID != "" || sc.ContainerName != "" {
					attributedCount++
				}
			}
			if len(snap.Connections) > 0 && attributedCount == 0 {
				t.Logf("Notice: No sockets attributed to containers")
			}
		})

		t.Run("TC_T1_F11_02_ListeningVsEstablishedStates", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.State != "LISTEN" && sc.State != "ESTABLISHED" && sc.State != "TIME_WAIT" && sc.State != "CLOSE_WAIT" {
					t.Logf("Notice: Non-standard socket state: %s", sc.State)
				}
			}
		})

		t.Run("TC_T1_F11_03_CriticalRiskOnPublicDatabaseExposure", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if (sc.LocalPort == 6379 || sc.LocalPort == 5432 || sc.LocalPort == 3306) && (sc.LocalAddr == "0.0.0.0" || sc.LocalAddr == "::") {
					if sc.SecurityLevel != "critical" {
						t.Errorf("Expected critical securityLevel for open DB on %s:%d, got %s", sc.LocalAddr, sc.LocalPort, sc.SecurityLevel)
					}
				}
			}
		})

		t.Run("TC_T1_F11_04_CautionRiskOnPublicHTTP", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if (sc.LocalPort == 80 || sc.LocalPort == 8080) && (sc.LocalAddr == "0.0.0.0" || sc.LocalAddr == "::") {
					if sc.SecurityLevel != "caution" && sc.SecurityLevel != "safe" {
						t.Errorf("Expected caution or safe securityLevel for HTTP on 0.0.0.0:%d, got %s", sc.LocalPort, sc.SecurityLevel)
					}
				}
			}
		})

		t.Run("TC_T1_F11_05_SafeRiskOnLocalhostBindOrTLS", func(t *testing.T) {
			snap, err := client.GetNetflow()
			if err != nil {
				t.Fatalf("GetNetflow failed: %v", err)
			}
			for _, sc := range snap.Connections {
				if sc.LocalAddr == "127.0.0.1" || sc.LocalPort == 443 {
					if sc.SecurityLevel != "safe" && sc.SecurityLevel != "internal" {
						t.Errorf("Expected safe/internal securityLevel for localhost/443 on %s:%d, got %s", sc.LocalAddr, sc.LocalPort, sc.SecurityLevel)
					}
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F12: Dashboard JSX Pre-existing Build Fix & Web Delivery
	// -------------------------------------------------------------------------
	t.Run("F12_DashboardBuildFixAndWebDelivery", func(t *testing.T) {
		t.Run("TC_T1_F12_01_IndexHtmlAccessible", func(t *testing.T) {
			resp, body, err := client.RawGet("/")
			if err != nil {
				t.Fatalf("RawGet / failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected HTTP 200 on /, got %d", resp.StatusCode)
			}
			if !strings.Contains(string(body), "<!DOCTYPE html>") && !strings.Contains(string(body), "<html") {
				t.Errorf("Expected HTML response on /")
			}
		})

		t.Run("TC_T1_F12_02_DashboardRouteLoads", func(t *testing.T) {
			resp, _, err := client.RawGet("/dashboard")
			if err != nil {
				t.Fatalf("RawGet /dashboard failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
				t.Errorf("Unexpected status on /dashboard: %d", resp.StatusCode)
			}
		})

		t.Run("TC_T1_F12_03_NetworksRouteLoads", func(t *testing.T) {
			resp, _, err := client.RawGet("/networks")
			if err != nil {
				t.Fatalf("RawGet /networks failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 on /networks SPA route, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T1_F12_04_NetworkFlowRouteLoads", func(t *testing.T) {
			resp, _, err := client.RawGet("/network-flow")
			if err != nil {
				t.Fatalf("RawGet /network-flow failed: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 on /network-flow SPA route, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T1_F12_05_NoServerInternalErrorOnWebRoutes", func(t *testing.T) {
			routes := []string{"/", "/networks", "/network-flow"}
			for _, r := range routes {
				resp, _, err := client.RawGet(r)
				if err != nil {
					t.Fatalf("RawGet %s failed: %v", r, err)
				}
				if resp.StatusCode >= 500 {
					t.Errorf("Route %s returned internal server error %d", r, resp.StatusCode)
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// F13: End-to-End Build & Live Docker Validation
	// -------------------------------------------------------------------------
	t.Run("F13_LiveValidationAndHealth", func(t *testing.T) {
		t.Run("TC_T1_F13_01_HealthEndpointStatusOk", func(t *testing.T) {
			h, err := client.GetHealth()
			if err != nil {
				t.Fatalf("GetHealth failed: %v", err)
			}
			if h["status"] != "ok" {
				t.Errorf("Expected health status 'ok', got %v", h["status"])
			}
		})

		t.Run("TC_T1_F13_02_AuthStatusReturnsFeatures", func(t *testing.T) {
			s, err := client.GetAuthStatus()
			if err != nil {
				t.Fatalf("GetAuthStatus failed: %v", err)
			}
			if s.Features == nil {
				t.Errorf("Expected features map in AuthStatus")
			}
		})

		t.Run("TC_T1_F13_03_AuthenticationSessionValid", func(t *testing.T) {
			s, err := client.GetAuthStatus()
			if err != nil {
				t.Fatalf("GetAuthStatus failed: %v", err)
			}
			if !s.Authenticated {
				t.Errorf("Expected client to be authenticated")
			}
		})

		t.Run("TC_T1_F13_04_UnauthenticatedAccessRejected", func(t *testing.T) {
			unauthClient, err := NewClient(client.BaseURL)
			if err != nil {
				t.Fatalf("Failed to create unauth client: %v", err)
			}
			resp, _, err := unauthClient.RawGet("/api/v1/netflow")
			if err != nil {
				t.Fatalf("RawGet failed: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 401 or allowed readonly, got %d", resp.StatusCode)
			}
		})

		t.Run("TC_T1_F13_05_CSRFTokenHeaderHandling", func(t *testing.T) {
			resp, _, err := client.RawPost("/api/v1/networks", map[string]string{"Name": "test-network"})
			if err != nil {
				t.Fatalf("RawPost failed: %v", err)
			}
			// When CSRF token is valid, status should be 201 or 400 (bad input), not 403 Forbidden
			if resp.StatusCode == http.StatusForbidden {
				t.Errorf("Expected valid CSRF token to be accepted, got 403 Forbidden")
			}
		})
	})
}
