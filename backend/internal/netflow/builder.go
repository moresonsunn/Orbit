package netflow

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/phili/orbit/internal/config"
	"github.com/phili/orbit/internal/docker"
	"github.com/phili/orbit/internal/model"
)

type Builder struct {
	cfg          *config.Config
	inv          *docker.Inventory
	streamer     *docker.StatsStreamer
	socketReader *SocketReader
}

func NewBuilder(cfg *config.Config, inv *docker.Inventory, streamer *docker.StatsStreamer) *Builder {
	return &Builder{
		cfg:          cfg,
		inv:          inv,
		streamer:     streamer,
		socketReader: NewSocketReader(cfg, inv),
	}
}

func (b *Builder) BuildSnapshot(ctx context.Context) NetflowSnapshot {
	now := time.Now()
	var containers []docker.Container
	var networks []network.Summary

	if b.inv != nil {
		containers = b.inv.Containers()
		func() {
			defer func() {
				_ = recover()
			}()
			if nets, err := b.inv.Networks(ctx, 30*time.Second); err == nil {
				networks = nets
			}
		}()
	}

	nodes := []TopologyNode{}
	edges := []TopologyEdge{}

	// 1. Host Node
	hostNode := TopologyNode{
		ID:    "host",
		Type:  NodeHost,
		Label: "Host-System",
		RxBps: model.NA(model.OK, ""),
		TxBps: model.NA(model.OK, ""),
	}
	nodes = append(nodes, hostNode)

	// 2. Network Nodes
	for _, n := range networks {
		netID := fmt.Sprintf("net-%s", n.ID)
		if len(n.ID) > 12 {
			netID = fmt.Sprintf("net-%s", n.ID[:12])
		}

		subnet := ""
		if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Subnet.IsValid() {
			subnet = n.IPAM.Config[0].Subnet.String()
		}

		nodes = append(nodes, TopologyNode{
			ID:     netID,
			Type:   NodeNetwork,
			Label:  n.Name,
			Subnet: subnet,
			RxBps:  model.NA(model.OK, ""),
			TxBps:  model.NA(model.OK, ""),
			Metadata: map[string]interface{}{
				"driver":   n.Driver,
				"scope":    n.Scope,
				"internal": n.Internal,
			},
		})

		// Edge from Host to Network
		edges = append(edges, TopologyEdge{
			ID:     fmt.Sprintf("edge-host-%s", netID),
			Source: "host",
			Target: netID,
			Type:   EdgeGateway,
			Active: true,
			RxBps:  model.NA(model.OK, ""),
			TxBps:  model.NA(model.OK, ""),
		})
	}

	// 3. Container Nodes & Connections
	containerStats := b.streamer.GetAllLatest()
	statsMap := make(map[string]docker.ContainerStats)
	for _, st := range containerStats {
		statsMap[st.ID] = st
	}

	traffic := []ContainerTrafficMetric{}
	var totalRxBps float64
	var totalTxBps float64

	for _, c := range containers {
		cNodeID := fmt.Sprintf("container-%s", c.ShortID)
		st, hasStats := statsMap[c.ID]

		rxGauge := model.NA(model.Unavailable, "no traffic metrics")
		txGauge := model.NA(model.Unavailable, "no traffic metrics")
		var rxBps, txBps, rxPps, txPps float64
		var rxBytes, txBytes uint64

		if hasStats {
			rxGauge = st.NetRxBps
			txGauge = st.NetTxBps
			if st.NetRxBps.Value != nil {
				rxBps = *st.NetRxBps.Value
			}
			if st.NetTxBps.Value != nil {
				txBps = *st.NetTxBps.Value
			}
			if st.NetRxBytes.Value != nil {
				rxBytes = uint64(*st.NetRxBytes.Value)
			}
			if st.NetTxBytes.Value != nil {
				txBytes = uint64(*st.NetTxBytes.Value)
			}
			if st.NetRxPps.Value != nil {
				rxPps = *st.NetRxPps.Value
			}
			if st.NetTxPps.Value != nil {
				txPps = *st.NetTxPps.Value
			}
		}

		totalRxBps += rxBps
		totalTxBps += txBps

		traffic = append(traffic, ContainerTrafficMetric{
			ContainerID:   c.ID,
			ContainerName: c.Name,
			Status:        c.State,
			RxBps:         rxBps,
			TxBps:         txBps,
			RxBytes:       rxBytes,
			TxBytes:       txBytes,
			RxPps:         rxPps,
			TxPps:         txPps,
		})

		nodes = append(nodes, TopologyNode{
			ID:     cNodeID,
			Type:   NodeContainer,
			Label:  c.Name,
			Status: c.State,
			RxBps:  rxGauge,
			TxBps:  txGauge,
			Metadata: map[string]interface{}{
				"image":        c.Image,
				"health":       c.Health,
				"restartCount": c.RestartCount,
			},
		})

		// Connect container to each Docker Network it belongs to
		for _, netRef := range c.Networks {
			netNodeID := fmt.Sprintf("net-%s", netRef.NetworkID)
			if len(netRef.NetworkID) > 12 {
				netNodeID = fmt.Sprintf("net-%s", netRef.NetworkID[:12])
			}

			active := false
			if rxGauge.Value != nil && *rxGauge.Value > 100 {
				active = true
			}
			if txGauge.Value != nil && *txGauge.Value > 100 {
				active = true
			}

			edges = append(edges, TopologyEdge{
				ID:     fmt.Sprintf("edge-%s-%s", netNodeID, cNodeID),
				Source: netNodeID,
				Target: cNodeID,
				Type:   EdgeMember,
				Active: active,
				RxBps:  rxGauge,
				TxBps:  txGauge,
			})
		}
	}

	sort.Slice(traffic, func(i, j int) bool {
		return (traffic[i].RxBps + traffic[i].TxBps) > (traffic[j].RxBps + traffic[j].TxBps)
	})

	// 4. Read sockets and generate protocol statistics
	conns := b.socketReader.ReadSockets()

	// Summarize protocols: configured vs observed
	protoMap := make(map[string]*ProtocolSummary)

	// Configured ports from container inspects
	for _, c := range containers {
		for _, p := range c.Ports {
			protoName := ClassifyPort(p.PrivatePort)
			if _, exists := protoMap[protoName]; !exists {
				protoMap[protoName] = &ProtocolSummary{Protocol: protoName}
			}
			protoMap[protoName].Configured++
		}
	}

	// Observed connections
	for _, conn := range conns {
		protoName := conn.AppProtocol
		if protoName == "" || protoName == "UNKNOWN" {
			protoName = strings.ToUpper(conn.Protocol)
		}
		if _, exists := protoMap[protoName]; !exists {
			protoMap[protoName] = &ProtocolSummary{Protocol: protoName}
		}
		protoMap[protoName].Observed++
		if conn.State == "ESTABLISHED" || conn.State == "ACTIVE" {
			protoMap[protoName].ActiveFlows++
		}
	}

	protocolStats := []ProtocolSummary{}
	for _, ps := range protoMap {
		protocolStats = append(protocolStats, *ps)
	}

	if conns == nil {
		conns = []SocketConnection{}
	}

	activeConns := 0
	securityAlerts := 0
	for _, conn := range conns {
		if conn.State == "ESTABLISHED" || conn.State == "ACTIVE" {
			activeConns++
		}
		if conn.SecurityLevel == "critical" || conn.SecurityLevel == "caution" {
			securityAlerts++
		}
	}

	totals := NetflowTotals{
		TotalRxBps:          totalRxBps,
		TotalTxBps:          totalTxBps,
		ActiveConnections:   activeConns,
		SecurityAlertsCount: securityAlerts,
	}

	return NetflowSnapshot{
		Timestamp:     now,
		EBPFAvailable: false,
		EBPFStatus:    "Standard Socket-Inspektion aktiv (eBPF-Erweiterung optional)",
		Topology: Topology{
			Timestamp: now,
			Nodes:     nodes,
			Edges:     edges,
		},
		ContainerTraffic: traffic,
		Connections:      conns,
		ProtocolStats:    protocolStats,
		Totals:           totals,
	}
}
