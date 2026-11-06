package netflow

import (
	"time"

	"github.com/phili/orbit/internal/model"
)

type NodeType string

const (
	NodeHost      NodeType = "host"
	NodeNetwork   NodeType = "network"
	NodeContainer NodeType = "container"
	NodeGateway   NodeType = "gateway"
	NodeExternal  NodeType = "external"
)

type EdgeType string

const (
	EdgeMember  EdgeType = "member"
	EdgeTraffic EdgeType = "traffic"
	EdgeGateway EdgeType = "gateway"
)

type TopologyNode struct {
	ID        string                 `json:"id"`
	Type      NodeType               `json:"type"`
	Label     string                 `json:"label"`
	Subnet    string                 `json:"subnet,omitempty"`
	IP        string                 `json:"ip,omitempty"`
	Status    string                 `json:"status,omitempty"` // running | stopped | healthy etc.
	RxBps     model.Gauge            `json:"rxBps"`
	TxBps     model.Gauge            `json:"txBps"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TopologyEdge struct {
	ID       string      `json:"id"`
	Source   string      `json:"source"`
	Target   string      `json:"target"`
	Type     EdgeType    `json:"type"`
	Active   bool        `json:"active"`
	RxBps    model.Gauge `json:"rxBps"`
	TxBps    model.Gauge `json:"txBps"`
	Protocol string      `json:"protocol,omitempty"`
}

type Topology struct {
	Timestamp time.Time      `json:"timestamp"`
	Nodes     []TopologyNode `json:"nodes"`
	Edges     []TopologyEdge `json:"edges"`
}

type SocketConnection struct {
	Protocol       string `json:"protocol"` // tcp, tcp6, udp, udp6
	LocalIP        string `json:"localIp"`
	LocalPort      uint16 `json:"localPort"`
	LocalAddr      string `json:"localAddr,omitempty"`
	RemoteIP       string `json:"remoteIp"`
	RemotePort     uint16 `json:"remotePort"`
	RemoteAddr     string `json:"remoteAddr,omitempty"`
	State          string `json:"state"` // LISTEN, ESTABLISHED, TIME_WAIT etc.
	ContainerID    string `json:"containerId,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
	ProcessName    string `json:"processName,omitempty"`
	AppProtocol    string `json:"appProtocol,omitempty"`    // HTTP, HTTPS, DNS, etc.
	DetectedProto  string `json:"detectedProto,omitempty"` // For compatibility: HTTP, HTTPS, etc.
	Classification string `json:"classification"`           // "port_based", "docker_inspection", or "ebpf_observed"
	DetectionMode  string `json:"detectionMode"`            // "socket_snapshot", "docker_engine", or "ebpf"
	SecurityLevel  string `json:"securityLevel"`            // "critical" | "caution" | "safe" | "internal"
	SecurityBadge  string `json:"securityBadge"`
	SecurityDesc   string `json:"securityDesc,omitempty"`
}

type ProtocolSummary struct {
	Protocol    string `json:"protocol"`
	Configured  int    `json:"configuredPortsCount"`
	Observed    int    `json:"observedConnectionsCount"`
	ActiveFlows int    `json:"activeFlows"`
}

type ContainerTrafficMetric struct {
	ContainerID   string  `json:"containerId"`
	ContainerName string  `json:"containerName"`
	Status        string  `json:"status,omitempty"`
	RxBps         float64 `json:"rxBps"`
	TxBps         float64 `json:"txBps"`
	RxBytes       uint64  `json:"rxBytes"`
	TxBytes       uint64  `json:"txBytes"`
	RxPps         float64 `json:"rxPps"`
	TxPps         float64 `json:"txPps"`
}

type NetflowTotals struct {
	TotalRxBps          float64 `json:"totalRxBps"`
	TotalTxBps          float64 `json:"totalTxBps"`
	ActiveConnections   int     `json:"activeConnections"`
	SecurityAlertsCount int     `json:"securityAlertsCount"`
}

type NetflowSnapshot struct {
	Timestamp        time.Time                `json:"timestamp"`
	EBPFAvailable    bool                     `json:"ebpfAvailable"`
	EBPFStatus       string                   `json:"ebpfStatus"`
	Topology         Topology                 `json:"topology"`
	ContainerTraffic []ContainerTrafficMetric `json:"containerTraffic"`
	Connections      []SocketConnection       `json:"connections"`
	ProtocolStats    []ProtocolSummary        `json:"protocolStats"`
	Totals           NetflowTotals            `json:"totals"`
}

