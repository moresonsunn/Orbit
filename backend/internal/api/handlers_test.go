package api

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/moby/moby/api/types/network"
)

func TestCalculateSubnetDetails(t *testing.T) {
	// Case 1: Standard /16 bridge network
	containers := map[string]NetworkContainerEndpoint{
		"c1": {
			Name:        "web",
			IPv4Address: "172.18.0.2/16",
			State:       "running",
		},
		"c2": {
			Name:        "db",
			IPv4Address: "172.18.0.3/16",
			State:       "running",
		},
		"c3": {
			Name:        "cache",
			IPv4Address: "172.18.0.4/16",
			State:       "running",
		},
	}

	details := calculateSubnetDetails("172.18.0.0/16", "172.18.0.1", containers, nil)
	if details == nil {
		t.Fatalf("expected non-nil SubnetDetails")
	}

	if details.Subnet != "172.18.0.0/16" {
		t.Errorf("Subnet = %q, want 172.18.0.0/16", details.Subnet)
	}
	if details.Gateway != "172.18.0.1" {
		t.Errorf("Gateway = %q, want 172.18.0.1", details.Gateway)
	}
	if details.NetworkIP != "172.18.0.0" {
		t.Errorf("NetworkIP = %q, want 172.18.0.0", details.NetworkIP)
	}
	if details.BroadcastIP != "172.18.255.255" {
		t.Errorf("BroadcastIP = %q, want 172.18.255.255", details.BroadcastIP)
	}
	if details.UsableRange != "172.18.0.1 - 172.18.255.254" {
		t.Errorf("UsableRange = %q, want 172.18.0.1 - 172.18.255.254", details.UsableRange)
	}
	if details.TotalIPs != 65536 {
		t.Errorf("TotalIPs = %d, want 65536", details.TotalIPs)
	}
	// 3 containers + 1 gateway = 4 used
	if details.UsedIPs != 4 {
		t.Errorf("UsedIPs = %d, want 4", details.UsedIPs)
	}
	if details.FreeIPs != 65532 {
		t.Errorf("FreeIPs = %d, want 65532", details.FreeIPs)
	}
	if details.UtilizationPercent <= 0 {
		t.Errorf("UtilizationPercent should be > 0, got %f", details.UtilizationPercent)
	}

	// Case 2: /24 subnet calculation
	details24 := calculateSubnetDetails("192.168.10.0/24", "192.168.10.1", nil, nil)
	if details24 == nil {
		t.Fatalf("expected non-nil SubnetDetails for /24")
	}
	if details24.NetworkIP != "192.168.10.0" {
		t.Errorf("NetworkIP = %q, want 192.168.10.0", details24.NetworkIP)
	}
	if details24.BroadcastIP != "192.168.10.255" {
		t.Errorf("BroadcastIP = %q, want 192.168.10.255", details24.BroadcastIP)
	}
	if details24.UsableRange != "192.168.10.1 - 192.168.10.254" {
		t.Errorf("UsableRange = %q, want 192.168.10.1 - 192.168.10.254", details24.UsableRange)
	}
	if details24.TotalIPs != 256 {
		t.Errorf("TotalIPs = %d, want 256", details24.TotalIPs)
	}

	// Case 3: Docker Inspect Status provided
	pfx := netip.MustParsePrefix("172.18.0.0/16")
	status := &network.Status{
		IPAM: network.IPAMStatus{
			Subnets: network.SubnetStatuses{
				pfx: network.SubnetStatus{
					IPsInUse:            8,
					DynamicIPsAvailable: 65528,
				},
			},
		},
	}
	detailsWithStatus := calculateSubnetDetails("172.18.0.0/16", "172.18.0.1", containers, status)
	if detailsWithStatus.UsedIPs != 8 {
		t.Errorf("UsedIPs from status = %d, want 8", detailsWithStatus.UsedIPs)
	}
	if detailsWithStatus.FreeIPs != 65528 {
		t.Errorf("FreeIPs from status = %d, want 65528", detailsWithStatus.FreeIPs)
	}
	if detailsWithStatus.TotalIPs != 65536 {
		t.Errorf("TotalIPs from status = %d, want 65536", detailsWithStatus.TotalIPs)
	}

	// Case 4: Invalid and empty inputs
	if res := calculateSubnetDetails("", "", nil, nil); res != nil {
		t.Errorf("expected nil for empty subnet, got %+v", res)
	}
	if res := calculateSubnetDetails("invalid-cidr", "", nil, nil); res != nil {
		t.Errorf("expected nil for invalid cidr, got %+v", res)
	}
}

func TestEnrichedDockerNetworkJSONSerialization(t *testing.T) {
	summary := network.Summary{
		Network: network.Network{
			ID:         "5edf60f8cc45e80442d3ee8cc586c119b01d076861c22d65a669e8df25f80684",
			Name:       "orbit_default",
			Driver:     "bridge",
			Scope:      "local",
			EnableIPv6: false,
			Internal:   false,
			Options: map[string]string{
				"com.docker.network.bridge.enable_icc": "true",
			},
			Labels: map[string]string{
				"com.docker.compose.project": "orbit",
			},
		},
	}

	enriched := EnrichedDockerNetwork{
		Summary: summary,
		SubnetDetails: &SubnetDetails{
			Subnet:             "172.18.0.0/16",
			Gateway:            "172.18.0.1",
			NetworkIP:          "172.18.0.0",
			BroadcastIP:        "172.18.255.255",
			UsableRange:        "172.18.0.1 - 172.18.255.254",
			TotalIPs:           65536,
			UsedIPs:            4,
			FreeIPs:            65532,
			UtilizationPercent: 0.01,
		},
		Containers: map[string]NetworkContainerEndpoint{
			"c12345": {
				Name:        "orbit-live",
				EndpointID:  "ep123",
				MacAddress:  "02:42:ac:12:00:02",
				IPv4Address: "172.18.0.2/16",
				IPv6Address: "",
				State:       "running",
			},
		},
	}

	data, err := json.Marshal(enriched)
	if err != nil {
		t.Fatalf("failed to marshal EnrichedDockerNetwork: %v", err)
	}

	// Verify standard fields are present at the root level (backward compatibility)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if raw["Id"] != summary.ID {
		t.Errorf("expected Id = %q, got %v", summary.ID, raw["Id"])
	}
	if raw["Name"] != summary.Name {
		t.Errorf("expected Name = %q, got %v", summary.Name, raw["Name"])
	}
	if raw["Driver"] != "bridge" {
		t.Errorf("expected Driver = 'bridge', got %v", raw["Driver"])
	}

	// Verify enriched SubnetDetails
	subDetailsRaw, ok := raw["SubnetDetails"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected SubnetDetails map in JSON")
	}
	if subDetailsRaw["Subnet"] != "172.18.0.0/16" {
		t.Errorf("expected SubnetDetails.Subnet = 172.18.0.0/16, got %v", subDetailsRaw["Subnet"])
	}
	if subDetailsRaw["NetworkIP"] != "172.18.0.0" {
		t.Errorf("expected SubnetDetails.NetworkIP = 172.18.0.0, got %v", subDetailsRaw["NetworkIP"])
	}

	// Verify enriched Containers
	containersRaw, ok := raw["Containers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Containers map in JSON")
	}
	cRaw, ok := containersRaw["c12345"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected container c12345 in Containers map")
	}
	if cRaw["Name"] != "orbit-live" {
		t.Errorf("expected container Name = 'orbit-live', got %v", cRaw["Name"])
	}
	if cRaw["State"] != "running" {
		t.Errorf("expected container State = 'running', got %v", cRaw["State"])
	}
}
