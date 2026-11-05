import React, { useState, useEffect } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  Position,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { Network, Activity, ShieldAlert, Radio, ArrowRight } from 'lucide-react';
import { NetflowSnapshot } from '../types';
import { api } from '../lib/api';
import { wsClient } from '../lib/ws';
import { formatBps } from '../lib/utils';

export const NetworkFlowPage: React.FC = () => {
  const [snapshot, setSnapshot] = useState<NetflowSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedConnection, setSelectedConnection] = useState<any | null>(null);

  const loadNetflow = async () => {
    try {
      const data = await api.getNetflow();
      setSnapshot(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadNetflow();

    // Subscribe to live netflow topic
    const unsub = wsClient.subscribe('netflow', (data: NetflowSnapshot) => {
      setSnapshot(data);
    });

    return () => unsub();
  }, []);

  if (loading || !snapshot) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 font-mono text-xs">
        Lade Netzwerktopologie und Verbindungen...
      </div>
    );
  }

  // Convert snapshot topology into React Flow nodes & edges with automatic layout
  const rawNodes = snapshot.topology.nodes;
  const rawEdges = snapshot.topology.edges;

  // Layout calculations
  const hostNodes = rawNodes.filter((n) => n.type === 'host');
  const networkNodes = rawNodes.filter((n) => n.type === 'network');
  const containerNodes = rawNodes.filter((n) => n.type === 'container');

  const nodes: Node[] = [
    // 1. Host Node
    ...hostNodes.map((n, i) => ({
      id: n.id,
      position: { x: 50, y: 150 + i * 180 },
      data: { label: n.label, type: n.type },
      style: {
        background: '#181818',
        color: '#ffffff',
        border: '1px solid #3b82f6',
        borderRadius: '6px',
        padding: '10px 18px',
        fontWeight: '600',
        fontSize: '12px',
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.4)',
      },
    })),

    // 2. Network Nodes
    ...networkNodes.map((n, i) => ({
      id: n.id,
      position: { x: 300, y: 60 + i * 140 },
      data: { label: `${n.label}\n${n.subnet || ''}`, type: n.type },
      style: {
        background: '#181818',
        color: '#a3a3a3',
        border: '1px solid #282828',
        borderRadius: '6px',
        padding: '8px 14px',
        fontSize: '11px',
        whiteSpace: 'pre-line',
      },
    })),

    // 3. Container Nodes
    ...containerNodes.map((n, i) => ({
      id: n.id,
      position: { x: 620, y: 40 + i * 75 },
      data: { label: n.label, type: n.type, status: n.status },
      style: {
        background: n.status === 'running' ? '#181818' : '#141414',
        color: n.status === 'running' ? '#10b981' : '#737373',
        border: n.status === 'running' ? '1px solid #059669' : '1px solid #282828',
        borderRadius: '6px',
        padding: '7px 14px',
        fontSize: '11px',
        fontWeight: '500',
      },
    })),
  ];

  const edges: Edge[] = rawEdges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    animated: e.active,
    style: {
      stroke: e.active ? '#3b82f6' : '#282828',
      strokeWidth: e.active ? 1.5 : 1,
    },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      color: e.active ? '#3b82f6' : '#282828',
    },
  }));

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Top Banner Notice */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-lg bg-[#181818] border border-[#282828]">
        <div>
          <h2 className="text-xl font-semibold text-white tracking-tight">Interaktive Netzwerktopologie</h2>
          <p className="text-xs text-neutral-400 mt-0.5">
            Host-System → Docker Networks → Container Verknüpfungen
          </p>
        </div>

        <div className="flex items-center gap-2 px-2.5 py-1 rounded-md bg-[#121212] border border-[#282828] text-xs text-neutral-300">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
          <span>{snapshot.ebpfStatus}</span>
        </div>
      </div>

      {/* React Flow Interactive Graph */}
      <div className="h-[460px] w-full rounded-lg overflow-hidden border border-[#282828] bg-[#121212] relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          minZoom={0.2}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
        >
          <Background color="#1f1f1f" gap={20} size={1} />
          <Controls position="top-left" showInteractive={false} />
          <MiniMap
            nodeStrokeColor="#3b82f6"
            nodeColor="#222222"
            maskColor="rgba(18, 18, 18, 0.85)"
            className="rounded-md border border-[#282828] bg-[#141414]"
          />
        </ReactFlow>
      </div>

      {/* Protocol Detection Summary */}
      <div className="p-4 rounded-lg bg-[#181818] border border-[#282828]">
        <h3 className="text-sm font-semibold text-white tracking-tight">Netzwerk-Protokoll-Statistik</h3>
        <p className="text-xs text-neutral-400 mt-0.5 mb-4">
          Unterscheidung zwischen konfigurierten Ports und tatsächlich beobachtetem Socket-Verkehr
        </p>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          {snapshot.protocolStats.map((ps) => (
            <div key={ps.protocol} className="p-3 bg-[#121212] rounded-md border border-[#242424]">
              <span className="text-xs font-semibold text-neutral-200 block truncate">{ps.protocol}</span>
              <div className="mt-2 space-y-1 text-[11px] font-mono">
                <div className="flex justify-between text-neutral-400">
                  <span>Konfiguriert:</span>
                  <span className="text-neutral-200 tabular-nums">{ps.configuredPortsCount}</span>
                </div>
                <div className="flex justify-between text-neutral-400">
                  <span>Beobachtet:</span>
                  <span className="text-emerald-400 font-medium tabular-nums">{ps.observedConnectionsCount}</span>
                </div>
                <div className="flex justify-between text-neutral-400">
                  <span>Aktiv:</span>
                  <span className="text-neutral-300 tabular-nums">{ps.activeFlows}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Observed Active Sockets Table (Direct on Canvas, Tailscale style) */}
      <div className="pt-2">
        <h3 className="text-base font-semibold text-white tracking-tight">
          Beobachtete Verbindungen ({snapshot.connections.length})
        </h3>
        <p className="text-xs text-neutral-400 mt-0.5 mb-3">
          Echte Socket-Tabellen aus Container- und Host-Netzwerk-Namespaces
        </p>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold">
                <th className="pb-3 px-3 font-semibold">Protokoll</th>
                <th className="pb-3 px-3 font-semibold">Lokale Adresse:Port</th>
                <th className="pb-3 px-3 font-semibold">Remote Adresse:Port</th>
                <th className="pb-3 px-3 font-semibold">Status</th>
                <th className="pb-3 px-3 font-semibold">Container</th>
                <th className="pb-3 px-3 font-semibold">Erkanntes Protokoll</th>
                <th className="pb-3 px-3 font-semibold text-right">Klassifikation</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1f1f1f]">
              {snapshot.connections.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-neutral-500 font-normal">
                    Keine aktiven Socket-Verbindungen erfasst.
                  </td>
                </tr>
              ) : (
                snapshot.connections.slice(0, 25).map((conn, i) => (
                  <tr key={i} className="hover:bg-white/[0.02] transition-colors">
                    <td className="py-2.5 px-3 text-neutral-300 font-mono text-xs">{conn.protocol}</td>
                    <td className="py-2.5 px-3 text-neutral-200 font-mono text-xs tabular-nums">
                      {conn.localIp}:{conn.localPort}
                    </td>
                    <td className="py-2.5 px-3 text-neutral-200 font-mono text-xs tabular-nums">
                      {conn.remoteIp}:{conn.remotePort}
                    </td>
                    <td className="py-2.5 px-3">
                      <span
                        className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-normal ${
                          conn.state === 'ESTABLISHED'
                            ? 'bg-emerald-950/40 text-emerald-400 border border-emerald-800/40'
                            : conn.state === 'LISTEN'
                            ? 'bg-neutral-800 text-neutral-400 border border-neutral-700'
                            : 'bg-amber-950/40 text-amber-400 border border-amber-800/40'
                        }`}
                      >
                        {conn.state}
                      </span>
                    </td>
                    <td className="py-2.5 px-3 text-white font-medium">{conn.containerName || '—'}</td>
                    <td className="py-2.5 px-3 text-neutral-300 font-mono text-xs">{conn.appProtocol || '—'}</td>
                    <td className="py-2.5 px-3 text-right text-neutral-500 text-[11px]">
                      {conn.classification === 'port_based' ? 'Port-basiert' : 'eBPF'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

