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
        background: '#151b26',
        color: '#60a5fa',
        border: '1.5px solid #3b82f6',
        borderRadius: '8px',
        padding: '10px 18px',
        fontWeight: '600',
        fontSize: '12px',
        boxShadow: '0 4px 14px rgba(59, 130, 246, 0.2)',
      },
    })),

    // 2. Network Nodes
    ...networkNodes.map((n, i) => ({
      id: n.id,
      position: { x: 300, y: 60 + i * 140 },
      data: { label: `${n.label}\n${n.subnet || ''}`, type: n.type },
      style: {
        background: '#151b26',
        color: '#93c5fd',
        border: '1px solid #334155',
        borderRadius: '8px',
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
        background: n.status === 'running' ? '#0f291e' : '#1e2430',
        color: n.status === 'running' ? '#34d399' : '#94a3b8',
        border: n.status === 'running' ? '1px solid #059669' : '1px solid #334155',
        borderRadius: '6px',
        padding: '7px 14px',
        fontSize: '11px',
      },
    })),
  ];

  const edges: Edge[] = rawEdges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    animated: e.active,
    style: {
      stroke: e.active ? '#3b82f6' : '#334155',
      strokeWidth: e.active ? 2 : 1,
    },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      color: e.active ? '#3b82f6' : '#334155',
    },
  }));

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      {/* Top Banner Notice */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400">
            <Network className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white tracking-tight">Interaktive Netzwerktopologie</h3>
            <p className="text-xs text-slate-400 font-normal mt-0.5">
              Host-System → Docker Networks → Container Verknüpfungen
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[#0e121b] border border-[#232d3f] text-xs font-medium text-slate-300">
          <span className="w-2 h-2 rounded-full bg-emerald-400" />
          <span>{snapshot.ebpfStatus}</span>
        </div>
      </div>

      {/* React Flow Interactive Graph */}
      <div className="h-[480px] w-full rounded-xl overflow-hidden border border-[#232d3f] bg-[#111622] relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          minZoom={0.2}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
        >
          <Background color="#1f2937" gap={18} size={1} />
          <Controls position="top-left" showInteractive={false} />
          <MiniMap
            nodeStrokeColor="#3b82f6"
            nodeColor="#1e293b"
            maskColor="rgba(14, 18, 27, 0.85)"
            className="rounded-lg border border-[#232d3f] bg-[#111622]"
          />
        </ReactFlow>
      </div>

      {/* Protocol Detection Summary */}
      <div className="p-5 rounded-xl bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-semibold text-white tracking-tight">Netzwerk-Protokoll-Statistik</h3>
            <p className="text-xs text-slate-400 font-normal mt-0.5">
              Unterscheidung zwischen konfigurierten Ports und tatsächlich beobachtetem Socket-Verkehr
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          {snapshot.protocolStats.map((ps) => (
            <div key={ps.protocol} className="p-3 bg-[#0e121b] rounded-lg border border-[#232d3f]">
              <span className="text-xs font-semibold text-blue-400 block truncate">{ps.protocol}</span>
              <div className="mt-2 space-y-1 text-[11px]">
                <div className="flex justify-between text-slate-400">
                  <span className="font-normal">Konfiguriert:</span>
                  <span className="text-slate-200 tabular-nums">{ps.configuredPortsCount}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span className="font-normal">Beobachtet:</span>
                  <span className="text-emerald-400 font-medium tabular-nums">{ps.observedConnectionsCount}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span className="font-normal">Aktiv:</span>
                  <span className="text-indigo-400 tabular-nums">{ps.activeFlows}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Observed Active Sockets Table */}
      <div className="p-5 rounded-xl bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-semibold text-white tracking-tight">
              Beobachtete Verbindungen ({snapshot.connections.length})
            </h3>
            <p className="text-xs text-slate-400 font-normal mt-0.5">
              Echte Socket-Tabellen aus Container- und Host-Netzwerk-Namespaces (Snapshot)
            </p>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-[#232d3f] text-slate-400 uppercase tracking-wider text-[11px] font-medium">
                <th className="pb-3 font-medium">Protokoll</th>
                <th className="pb-3 font-medium">Lokale Adresse:Port</th>
                <th className="pb-3 font-medium">Remote Adresse:Port</th>
                <th className="pb-3 font-medium">Status</th>
                <th className="pb-3 font-medium">Container</th>
                <th className="pb-3 font-medium">Erkanntes Protokoll</th>
                <th className="pb-3 font-medium text-right">Klassifikation</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#232d3f]/60">
              {snapshot.connections.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-6 text-center text-slate-400 font-normal">
                    Keine aktiven Socket-Verbindungen erfasst.
                  </td>
                </tr>
              ) : (
                snapshot.connections.slice(0, 25).map((conn, i) => (
                  <tr key={i} className="hover:bg-[#1a2230]/50 transition-colors">
                    <td className="py-2.5 text-blue-400 font-semibold">{conn.protocol}</td>
                    <td className="py-2.5 text-slate-200 tabular-nums">
                      {conn.localIp}:{conn.localPort}
                    </td>
                    <td className="py-2.5 text-slate-200 tabular-nums">
                      {conn.remoteIp}:{conn.remotePort}
                    </td>
                    <td className="py-2.5">
                      <span
                        className={`inline-block px-2 py-0.5 rounded text-[10px] font-medium ${
                          conn.state === 'ESTABLISHED'
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : conn.state === 'LISTEN'
                            ? 'bg-slate-800 text-slate-300 border border-slate-700'
                            : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                        }`}
                      >
                        {conn.state}
                      </span>
                    </td>
                    <td className="py-2.5 text-white font-medium">{conn.containerName || '—'}</td>
                    <td className="py-2.5 text-indigo-400 font-medium">{conn.appProtocol || '—'}</td>
                    <td className="py-2.5 text-right text-slate-400 text-[10px]">
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

