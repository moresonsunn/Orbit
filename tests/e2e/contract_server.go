package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// ContractServer provides a reference mock server adhering strictly to Orbit's API specifications.
type ContractServer struct {
	Server    *httptest.Server
	mu        sync.Mutex
	Networks  map[string]EnrichedDockerNetwork
	Sessions  map[string]string // cookie token -> user
	CSRFTokens map[string]bool
}

// NewContractServer creates and launches a reference ContractServer.
func NewContractServer() *ContractServer {
	cs := &ContractServer{
		Networks:   make(map[string]EnrichedDockerNetwork),
		Sessions:   make(map[string]string),
		CSRFTokens: make(map[string]bool),
	}

	cs.populateDefaultNetworks()

	mux := http.NewServeMux()

	// Frontend SPA Routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Orbit Console</title><script type="module" src="/assets/index.js"></script></head><body><div id="root">Orbit App</div></body></html>`))
	})

	mux.HandleFunc("/assets/index.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`console.log("Orbit Frontend Bundle Loaded");`))
	})

	// Health API
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	// Auth API
	mux.HandleFunc("/api/v1/auth/status", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("orbit_session")
		loggedIn := false
		var user map[string]interface{}
		var csrfToken string

		if err == nil && cookie != nil {
			cs.mu.Lock()
			if u, ok := cs.Sessions[cookie.Value]; ok {
				loggedIn = true
				user = map[string]interface{}{"id": 1, "username": u, "role": "admin"}
				csrfToken = "test-csrf-token-" + cookie.Value
			}
			cs.mu.Unlock()
		}

		res := map[string]interface{}{
			"needsSetup":    false,
			"authenticated": loggedIn,
			"features": map[string]interface{}{
				"enableActions":     true,
				"enableDestructive": true,
				"ebpfMode":          "auto",
				"gpuMode":           "auto",
			},
		}
		if loggedIn {
			res["user"] = user
			res["csrfToken"] = csrfToken
		}
		writeJSONResponse(w, http.StatusOK, res)
	})

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "ungültiger Request Body")
			return
		}

		if req.Username != "admin" || req.Password != "OrbitAdminSecret2026!" {
			writeErrorResponse(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
			return
		}

		token := fmt.Sprintf("session-%d", time.Now().UnixNano())
		csrf := fmt.Sprintf("csrf-%d", time.Now().UnixNano())

		cs.mu.Lock()
		cs.Sessions[token] = req.Username
		cs.CSRFTokens[csrf] = true
		cs.mu.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "orbit_session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
		})

		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"csrfToken": csrf,
			"user": map[string]interface{}{
				"id":       1,
				"username": req.Username,
				"role":     "admin",
			},
		})
	})

	// Networks API
	mux.HandleFunc("/api/v1/networks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			cs.mu.Lock()
			var list []EnrichedDockerNetwork
			for _, n := range cs.Networks {
				list = append(list, n)
			}
			cs.mu.Unlock()
			writeJSONResponse(w, http.StatusOK, list)
			return
		}
		if r.Method == http.MethodPost {
			// Require auth & CSRF
			if !cs.checkCSRF(r) {
				writeErrorResponse(w, http.StatusForbidden, "CSRF token missing or invalid")
				return
			}
			var newNet EnrichedDockerNetwork
			if err := json.NewDecoder(r.Body).Decode(&newNet); err != nil {
				writeErrorResponse(w, http.StatusBadRequest, "Invalid network configuration")
				return
			}
			if newNet.Id == "" {
				newNet.Id = fmt.Sprintf("net-%d", time.Now().UnixNano())
			}
			if newNet.Containers == nil {
				newNet.Containers = make(map[string]NetworkContainerMember)
			}
			cs.mu.Lock()
			cs.Networks[newNet.Id] = newNet
			cs.mu.Unlock()
			writeJSONResponse(w, http.StatusCreated, newNet)
			return
		}
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	mux.HandleFunc("/api/v1/networks/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/networks/")
		if id == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Network ID required")
			return
		}

		if r.Method == http.MethodDelete && !cs.checkCSRF(r) {
			writeErrorResponse(w, http.StatusForbidden, "CSRF token missing or invalid")
			return
		}

		cs.mu.Lock()
		defer cs.mu.Unlock()

		net, ok := cs.Networks[id]
		if !ok {
			// Also search by prefix or name
			for _, n := range cs.Networks {
				if strings.HasPrefix(n.Id, id) || n.Name == id {
					net = n
					ok = true
					break
				}
			}
		}

		if !ok {
			writeErrorResponse(w, http.StatusNotFound, "Netzwerk nicht gefunden")
			return
		}

		if r.Method == http.MethodGet {
			writeJSONResponse(w, http.StatusOK, net)
			return
		}

		if r.Method == http.MethodDelete {
			delete(cs.Networks, net.Id)
			writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Netzwerk gelöscht"})
			return
		}

		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	// Netflow API
	mux.HandleFunc("/api/v1/netflow", func(w http.ResponseWriter, r *http.Request) {
		snapshot := cs.buildEnrichedSnapshot()
		writeJSONResponse(w, http.StatusOK, snapshot)
	})

	// Containers API
	mux.HandleFunc("/api/v1/containers", func(w http.ResponseWriter, r *http.Request) {
		containers := []ContainerSummary{
			{ID: "c169e342f941", Name: "test-orbit-healthy", State: "running", HealthStatus: "healthy", ComposeProject: "fixtures"},
			{ID: "0690d0622806", Name: "test-orbit-cpu-load", State: "running", ComposeProject: "fixtures"},
			{ID: "731c8504e958", Name: "test-orbit-mem-load", State: "running", ComposeProject: "fixtures"},
			{ID: "3ce4698ab9f5", Name: "test-orbit-traffic", State: "running", ComposeProject: "fixtures"},
			{ID: "0bb81da6f1e9", Name: "test-orbit-unhealthy", State: "running", HealthStatus: "unhealthy", ComposeProject: "fixtures"},
			{ID: "9cb372a6977d", Name: "orbit-live", State: "running"},
		}
		writeJSONResponse(w, http.StatusOK, containers)
	})

	cs.Server = httptest.NewServer(mux)
	return cs
}

func (cs *ContractServer) Close() {
	cs.Server.Close()
}

func (cs *ContractServer) URL() string {
	return cs.Server.URL
}

func (cs *ContractServer) checkCSRF(r *http.Request) bool {
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf == "" {
		return false
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.CSRFTokens[csrf] || strings.HasPrefix(csrf, "test-csrf-token-")
}

func (cs *ContractServer) populateDefaultNetworks() {
	cs.Networks["c592a2137b32"] = EnrichedDockerNetwork{
		Id:         "c592a2137b32",
		Name:       "bridge",
		Driver:     "bridge",
		Scope:      "local",
		Internal:   false,
		EnableIPv6: false,
		IPAM: IPAMData{
			Driver: "default",
			Config: []IPAMConfig{
				{Subnet: "172.17.0.0/16", Gateway: "172.17.0.1"},
			},
		},
		SubnetDetails: &SubnetDetails{
			Subnet:             "172.17.0.0/16",
			Gateway:            "172.17.0.1",
			NetworkIP:          "172.17.0.0",
			BroadcastIP:        "172.17.255.255",
			UsableRange:        "172.17.0.1 - 172.17.255.254",
			TotalIPs:           65536,
			UsedIPs:            1,
			FreeIPs:            65533,
			UtilizationPercent: 0.0015,
		},
		Containers: map[string]NetworkContainerMember{
			"9cb372a6977d": {
				Name:        "orbit-live",
				EndpointID:  "ep-orbit-bridge",
				MacAddress:  "02:42:ac:11:00:02",
				IPv4Address: "172.17.0.2",
				IPv6Address: "",
				State:       "running",
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.default_bridge":       "true",
			"com.docker.network.bridge.enable_icc":           "true",
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.name":                 "docker0",
		},
	}

	cs.Networks["5edf60f8cc45"] = EnrichedDockerNetwork{
		Id:         "5edf60f8cc45",
		Name:       "fixtures_default",
		Driver:     "bridge",
		Scope:      "local",
		Internal:   false,
		EnableIPv6: false,
		IPAM: IPAMData{
			Driver: "default",
			Config: []IPAMConfig{
				{Subnet: "172.18.0.0/16", Gateway: "172.18.0.1"},
			},
		},
		SubnetDetails: &SubnetDetails{
			Subnet:             "172.18.0.0/16",
			Gateway:            "172.18.0.1",
			NetworkIP:          "172.18.0.0",
			BroadcastIP:        "172.18.255.255",
			UsableRange:        "172.18.0.1 - 172.18.255.254",
			TotalIPs:           65536,
			UsedIPs:            5,
			FreeIPs:            65529,
			UtilizationPercent: 0.0076,
		},
		Containers: map[string]NetworkContainerMember{
			"c169e342f941": {
				Name:        "test-orbit-healthy",
				EndpointID:  "ep-healthy",
				MacAddress:  "02:42:ac:12:00:02",
				IPv4Address: "172.18.0.2",
				IPv6Address: "",
				State:       "running",
			},
			"0690d0622806": {
				Name:        "test-orbit-cpu-load",
				EndpointID:  "ep-cpu",
				MacAddress:  "02:42:ac:12:00:03",
				IPv4Address: "172.18.0.3",
				IPv6Address: "",
				State:       "running",
			},
			"731c8504e958": {
				Name:        "test-orbit-mem-load",
				EndpointID:  "ep-mem",
				MacAddress:  "02:42:ac:12:00:04",
				IPv4Address: "172.18.0.4",
				IPv6Address: "",
				State:       "running",
			},
			"3ce4698ab9f5": {
				Name:        "test-orbit-traffic",
				EndpointID:  "ep-traffic",
				MacAddress:  "02:42:ac:12:00:05",
				IPv4Address: "172.18.0.5",
				IPv6Address: "",
				State:       "running",
			},
			"0bb81da6f1e9": {
				Name:        "test-orbit-unhealthy",
				EndpointID:  "ep-unhealthy",
				MacAddress:  "02:42:ac:12:00:06",
				IPv4Address: "172.18.0.6",
				IPv6Address: "",
				State:       "running",
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "true",
			"com.docker.network.driver.mtu":        "1500",
		},
		Labels: map[string]string{
			"com.docker.compose.network": "default",
			"com.docker.compose.project": "fixtures",
		},
	}

	cs.Networks["131f74918e88"] = EnrichedDockerNetwork{
		Id:         "131f74918e88",
		Name:       "host",
		Driver:     "host",
		Scope:      "local",
		Internal:   false,
		EnableIPv6: false,
		IPAM:       IPAMData{Driver: "default"},
		Containers: make(map[string]NetworkContainerMember),
	}

	cs.Networks["55b2fb090708"] = EnrichedDockerNetwork{
		Id:         "55b2fb090708",
		Name:       "none",
		Driver:     "null",
		Scope:      "local",
		Internal:   false,
		EnableIPv6: false,
		IPAM:       IPAMData{Driver: "default"},
		Containers: make(map[string]NetworkContainerMember),
	}
}

func (cs *ContractServer) buildEnrichedSnapshot() EnrichedNetflowSnapshot {
	return EnrichedNetflowSnapshot{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		EBPFAvailable: false,
		EBPFStatus:    "disabled",
		Topology: Topology{
			Nodes: []TopologyNode{
				{ID: "host", Type: "host", Label: "Docker Host"},
				{ID: "net-fixtures", Type: "network", Label: "fixtures_default", Subnet: "172.18.0.0/16"},
				{ID: "c169e342f941", Type: "container", Label: "test-orbit-healthy", IP: "172.18.0.2", Status: "running"},
				{ID: "3ce4698ab9f5", Type: "container", Label: "test-orbit-traffic", IP: "172.18.0.5", Status: "running"},
				{ID: "0690d0622806", Type: "container", Label: "test-orbit-cpu-load", IP: "172.18.0.3", Status: "running"},
			},
			Edges: []TopologyEdge{
				{ID: "e1", Source: "net-fixtures", Target: "c169e342f941", Type: "member", Active: true},
				{ID: "e2", Source: "net-fixtures", Target: "3ce4698ab9f5", Type: "member", Active: true},
				{ID: "e3", Source: "c169e342f941", Target: "3ce4698ab9f5", Type: "traffic", Active: true, Protocol: "HTTP"},
			},
		},
		ContainerTraffic: []ContainerTrafficMetric{
			{ContainerID: "3ce4698ab9f5", ContainerName: "test-orbit-traffic", RxBps: 125000, TxBps: 240000, RxBytes: 15400000, TxBytes: 29500000, RxPps: 145, TxPps: 210},
			{ContainerID: "c169e342f941", ContainerName: "test-orbit-healthy", RxBps: 4500, TxBps: 3200, RxBytes: 850000, TxBytes: 620000, RxPps: 12, TxPps: 8},
			{ContainerID: "0690d0622806", ContainerName: "test-orbit-cpu-load", RxBps: 800, TxBps: 400, RxBytes: 120000, TxBytes: 45000, RxPps: 2, TxPps: 1},
			{ContainerID: "731c8504e958", ContainerName: "test-orbit-mem-load", RxBps: 0, TxBps: 0, RxBytes: 50000, TxBytes: 20000, RxPps: 0, TxPps: 0},
			{ContainerID: "0bb81da6f1e9", ContainerName: "test-orbit-unhealthy", RxBps: 0, TxBps: 0, RxBytes: 10000, TxBytes: 5000, RxPps: 0, TxPps: 0},
			{ContainerID: "9cb372a6977d", ContainerName: "orbit-live", RxBps: 1200, TxBps: 8500, RxBytes: 320000, TxBytes: 1420000, RxPps: 15, TxPps: 28},
		},
		Connections: []SocketConnection{
			{
				LocalAddr:      "0.0.0.0",
				LocalPort:      6379,
				RemoteAddr:     "0.0.0.0",
				RemotePort:     0,
				Protocol:       "tcp",
				State:          "LISTEN",
				ContainerID:    "redis-container-id",
				ContainerName:  "test-redis",
				ProcessName:    "redis-server",
				Classification: "port_based",
				DetectedProto:  "REDIS",
				SecurityLevel:  "critical",
				SecurityBadge:  "CRITICAL",
				SecurityDesc:   "Unverschlüsselte Datenbank auf öffentlicher Schnittstelle (0.0.0.0) exponiert",
			},
			{
				LocalAddr:      "0.0.0.0",
				LocalPort:      80,
				RemoteAddr:     "0.0.0.0",
				RemotePort:     0,
				Protocol:       "tcp",
				State:          "LISTEN",
				ContainerID:    "3ce4698ab9f5",
				ContainerName:  "test-orbit-traffic",
				ProcessName:    "nginx",
				Classification: "port_based",
				DetectedProto:  "HTTP",
				SecurityLevel:  "caution",
				SecurityBadge:  "CAUTION",
				SecurityDesc:   "Unverschlüsselter HTTP-Webdienst auf öffentlicher Schnittstelle exponiert",
			},
			{
				LocalAddr:      "127.0.0.1",
				LocalPort:      5432,
				RemoteAddr:     "0.0.0.0",
				RemotePort:     0,
				Protocol:       "tcp",
				State:          "LISTEN",
				ContainerID:    "postgres-container-id",
				ContainerName:  "test-postgres",
				ProcessName:    "postgres",
				Classification: "port_based",
				DetectedProto:  "POSTGRES",
				SecurityLevel:  "safe",
				SecurityBadge:  "SAFE",
				SecurityDesc:   "Dienst auf Loopback-Schnittstelle (127.0.0.1) gebunden",
			},
			{
				LocalAddr:      "172.18.0.2",
				LocalPort:      8080,
				RemoteAddr:     "0.0.0.0",
				RemotePort:     0,
				Protocol:       "tcp",
				State:          "LISTEN",
				ContainerID:    "c169e342f941",
				ContainerName:  "test-orbit-healthy",
				ProcessName:    "app",
				Classification: "port_based",
				DetectedProto:  "HTTP",
				SecurityLevel:  "internal",
				SecurityBadge:  "INTERNAL",
				SecurityDesc:   "Internes Container-Netzwerk ohne öffentlichen Port-Bind",
			},
			{
				LocalAddr:      "0.0.0.0",
				LocalPort:      443,
				RemoteAddr:     "0.0.0.0",
				RemotePort:     0,
				Protocol:       "tcp",
				State:          "LISTEN",
				ContainerID:    "web-tls-container-id",
				ContainerName:  "test-web-tls",
				ProcessName:    "caddy",
				Classification: "port_based",
				DetectedProto:  "HTTPS",
				SecurityLevel:  "safe",
				SecurityBadge:  "SAFE",
				SecurityDesc:   "Verschlüsselter TLS/HTTPS Dienst auf Port 443",
			},
		},
		ProtocolStats: []ProtocolSummary{
			{Protocol: "HTTP", Configured: 2, Observed: 4, Active: 2},
			{Protocol: "HTTPS", Configured: 1, Observed: 2, Active: 1},
			{Protocol: "REDIS", Configured: 1, Observed: 1, Active: 1},
			{Protocol: "POSTGRES", Configured: 1, Observed: 1, Active: 1},
			{Protocol: "DNS", Configured: 1, Observed: 0, Active: 0},
			{Protocol: "SSH", Configured: 0, Observed: 0, Active: 0},
		},
		Totals: NetflowTotals{
			TotalRxBps:          131500,
			TotalTxBps:          252100,
			ActiveConnections:   5,
			SecurityAlertsCount: 2,
		},
	}
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, map[string]string{"error": message})
}
