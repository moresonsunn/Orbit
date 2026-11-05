package netflow

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	containers := b.inv.Containers()
	networks, _ := b.inv.Networks(ctx, 30*time.Second)

	var nodes []TopologyNode
	var edges []TopologyEdge

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

	for _, c := range containers {
		cNodeID := fmt.Sprintf("container-%s", c.ShortID)
		st, hasStats := statsMap[c.ID]

		rxGauge := model.NA(model.Unavailable, "no traffic metrics")
		txGauge := model.NA(model.Unavailable, "no traffic metrics")
		if hasStats {
			rxGauge = st.NetRxBps
			txGauge = st.NetTxBps
		}

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

	var protocolStats []ProtocolSummary
	for _, ps := range protoMap {
		protocolStats = append(protocolStats, *ps)
	}

	return NetflowSnapshot{
		Timestamp: now,
		EBPFAvailable: false,
		EBPFStatus: "Standard Socket-Inspektion aktiv (eBPF-Erweiterung optional)",
		Topology: Topology{
			Timestamp: now,
			Nodes:     nodes,
			Edges:     edges,
		},
		Connections:   conns,
		ProtocolStats: protocolStats,
	}
}
