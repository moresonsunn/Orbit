package e2e

// IPAMConfig describes an IPAM configuration block in Docker.
type IPAMConfig struct {
	Subnet  string `json:"Subnet"`
	Gateway string `json:"Gateway"`
	IPRange string `json:"IPRange,omitempty"`
}

// IPAMData holds IPAM driver and configs.
type IPAMData struct {
	Driver string       `json:"Driver"`
	Config []IPAMConfig `json:"Config"`
}

// SubnetDetails describes the calculated IPAM subnet metrics.
type SubnetDetails struct {
	Subnet             string  `json:"Subnet"`
	Gateway            string  `json:"Gateway"`
	NetworkIP          string  `json:"NetworkIP"`
	BroadcastIP        string  `json:"BroadcastIP"`
	UsableRange        string  `json:"UsableRange"`
	TotalIPs           int     `json:"TotalIPs"`
	UsedIPs            int     `json:"UsedIPs"`
	FreeIPs            int     `json:"FreeIPs"`
	UtilizationPercent float64 `json:"UtilizationPercent"`
}

// NetworkContainerMember describes a connected container attachment.
type NetworkContainerMember struct {
	Name        string `json:"Name"`
	EndpointID  string `json:"EndpointID"`
	MacAddress  string `json:"MacAddress"`
	IPv4Address string `json:"IPv4Address"`
	IPv6Address string `json:"IPv6Address"`
	State       string `json:"State,omitempty"`
}

// EnrichedDockerNetwork matches the enriched networks interface contract.
type EnrichedDockerNetwork struct {
	Id            string                            `json:"Id"`
	Name          string                            `json:"Name"`
	Driver        string                            `json:"Driver"`
	Scope         string                            `json:"Scope"`
	Internal      bool                              `json:"Internal"`
	EnableIPv6    bool                              `json:"EnableIPv6"`
	IPAM          IPAMData                          `json:"IPAM"`
	SubnetDetails *SubnetDetails                    `json:"SubnetDetails,omitempty"`
	Containers    map[string]NetworkContainerMember `json:"Containers"`
	Options       map[string]string                 `json:"Options,omitempty"`
	Labels        map[string]string                 `json:"Labels,omitempty"`
}

// TopologyNode represents a node in the interactive topology graph.
type TopologyNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // host, network, container, gateway, external
	Label    string                 `json:"label"`
	Subnet   string                 `json:"subnet,omitempty"`
	IP       string                 `json:"ip,omitempty"`
	Status   string                 `json:"status,omitempty"`
	RxBps    interface{}            `json:"rxBps"`
	TxBps    interface{}            `json:"txBps"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TopologyEdge represents a link between nodes.
type TopologyEdge struct {
	ID       string      `json:"id"`
	Source   string      `json:"source"`
	Target   string      `json:"target"`
	Type     string      `json:"type"` // member, traffic, gateway
	Active   bool        `json:"active"`
	RxBps    interface{} `json:"rxBps"`
	TxBps    interface{} `json:"txBps"`
	Protocol string      `json:"protocol,omitempty"`
}

// Topology holds nodes and edges.
type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// ContainerTrafficMetric captures per-container live bandwidth rates.
type ContainerTrafficMetric struct {
	ContainerID   string  `json:"containerId"`
	ContainerName string  `json:"containerName"`
	RxBps         float64 `json:"rxBps"`
	TxBps         float64 `json:"txBps"`
	RxBytes       uint64  `json:"rxBytes"`
	TxBytes       uint64  `json:"txBytes"`
	RxPps         float64 `json:"rxPps"`
	TxPps         float64 `json:"txPps"`
}

// SocketConnection captures synthesized and procfs socket metrics.
type SocketConnection struct {
	LocalAddr      string `json:"localAddr"`
	LocalPort      uint16 `json:"localPort"`
	RemoteAddr     string `json:"remoteAddr"`
	RemotePort     uint16 `json:"remotePort"`
	Protocol       string `json:"protocol"`
	State          string `json:"state"`
	ContainerID    string `json:"containerId"`
	ContainerName  string `json:"containerName"`
	ProcessName    string `json:"processName,omitempty"`
	Classification string `json:"classification"`
	DetectedProto  string `json:"detectedProto"`
	SecurityLevel  string `json:"securityLevel"` // critical, caution, safe, internal
	SecurityBadge  string `json:"securityBadge"`
	SecurityDesc   string `json:"securityDesc"`
}

// ProtocolSummary captures protocol stats counts.
type ProtocolSummary struct {
	Protocol    string `json:"protocol"`
	Configured  int    `json:"configured"`
	Observed    int    `json:"observed"`
	Active      int    `json:"active"`
}

// NetflowTotals captures aggregated bandwidth and alerts.
type NetflowTotals struct {
	TotalRxBps          float64 `json:"totalRxBps"`
	TotalTxBps          float64 `json:"totalTxBps"`
	ActiveConnections   int     `json:"activeConnections"`
	SecurityAlertsCount int     `json:"securityAlertsCount"`
}

// EnrichedNetflowSnapshot matches the enriched netflow interface contract.
type EnrichedNetflowSnapshot struct {
	Timestamp        string                   `json:"timestamp"`
	EBPFAvailable    bool                     `json:"ebpfAvailable,omitempty"`
	EBPFStatus       string                   `json:"ebpfStatus,omitempty"`
	Topology         Topology                 `json:"topology"`
	ContainerTraffic []ContainerTrafficMetric `json:"containerTraffic"`
	Connections      []SocketConnection       `json:"connections"`
	ProtocolStats    []ProtocolSummary        `json:"protocolStats"`
	Totals           NetflowTotals            `json:"totals"`
}

// AuthStatus represents the authentication status response.
type AuthStatus struct {
	Authenticated bool                   `json:"authenticated"`
	NeedsSetup    bool                   `json:"needsSetup"`
	Features      map[string]interface{} `json:"features"`
	User          map[string]interface{} `json:"user,omitempty"`
	CSRFToken     string                 `json:"csrfToken,omitempty"`
}

// LoginResponse represents the login response body.
type LoginResponse struct {
	CSRFToken string                 `json:"csrfToken"`
	User      map[string]interface{} `json:"user"`
	Error     string                 `json:"error,omitempty"`
}

// ContainerSummary represents a container item from /api/v1/containers.
type ContainerSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	State          string `json:"state"`
	HealthStatus   string `json:"healthStatus,omitempty"`
	ComposeProject string `json:"composeProject,omitempty"`
}
