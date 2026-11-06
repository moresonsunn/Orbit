/// <reference types="vite/client" />
import React, { useState, useEffect, useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {
  Network,
  Activity,
  ShieldAlert,
  ShieldCheck,
  Radio,
  ArrowRight,
  ArrowDown,
  ArrowUp,
  AlertTriangle,
  AlertCircle,
  Lock,
  Search,
  Filter,
  Copy,
  Check,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  RefreshCw,
  X,
  Boxes,
  Layers,
  Server,
  Globe,
  SlidersHorizontal,
} from 'lucide-react';
import {
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Tooltip,
} from 'recharts';
import { NetflowSnapshot, SocketConnection, ContainerTrafficMetric, ProtocolSummary } from '../types';
import { api } from '../lib/api';
import { wsClient } from '../lib/ws';
import { formatBps, formatBytes, formatPercent, cn } from '../lib/utils';
import { StatusBadge } from '../components/common/StatusBadge';

type SecurityRiskLevel = 'critical' | 'caution' | 'safe' | 'internal';
type FilterProtocol = 'all' | 'tcp' | 'udp';
type FilterState = 'all' | 'ESTABLISHED' | 'LISTEN' | 'TIME_WAIT';
type FilterSecurity = 'all' | SecurityRiskLevel;

function getSecurityLevel(conn: SocketConnection): SecurityRiskLevel {
  if (conn.securityLevel) {
    return conn.securityLevel;
  }
  const local = (conn.localAddr || conn.localIp || '').toLowerCase();
  const port = conn.localPort;
  const isPublic = local.startsWith('0.0.0.0') || local.startsWith('::') || local.startsWith('[::]');
  const isLoopback = local.startsWith('127.') || local.startsWith('::1') || local.startsWith('localhost');

  if (isLoopback) return 'safe';
  if (isPublic) {
    const criticalPorts = [21, 23, 2375, 3306, 5432, 6379, 9200, 11211, 27017];
    if (criticalPorts.includes(port)) return 'critical';
    const cautionPorts = [80, 8080, 8000, 8888, 3000];
    if (cautionPorts.includes(port)) return 'caution';
    if (port === 443 || port === 22 || port === 2376) return 'safe';
    return 'caution';
  }
  return 'internal';
}

function getSecurityMeta(level: SecurityRiskLevel, customDesc?: string) {
  switch (level) {
    case 'critical':
      return {
        label: 'Kritisch',
        fullLabel: 'Kritisch: Ungesicherter öffentlicher Dienst',
        desc: customDesc || 'Unverschlüsselter Dienst / Datenbank öffentlich an 0.0.0.0 gebunden',
        badgeClass: 'bg-rose-950/40 text-rose-400 border border-rose-800/50',
        icon: AlertTriangle,
        iconColor: 'text-rose-400',
      };
    case 'caution':
      return {
        label: 'Warnung',
        fullLabel: 'Warnung: Öffentliche Schnittstelle',
        desc: customDesc || 'Öffentlich erreichbarer Port / Plaintext HTTP an 0.0.0.0 gebunden',
        badgeClass: 'bg-amber-950/40 text-amber-400 border border-amber-800/50',
        icon: AlertCircle,
        iconColor: 'text-amber-400',
      };
    case 'safe':
      return {
        label: 'Sicher',
        fullLabel: 'Sicher: Verschlüsselt oder Loopback',
        desc: customDesc || 'An Host-Loopback (127.0.0.1) gebunden oder TLS-verschlüsselt',
        badgeClass: 'bg-emerald-950/40 text-emerald-400 border border-emerald-800/50',
        icon: ShieldCheck,
        iconColor: 'text-emerald-400',
      };
    case 'internal':
    default:
      return {
        label: 'Intern',
        fullLabel: 'Intern: Privates Container-Netzwerk',
        desc: customDesc || 'Verbindung innerhalb des privaten Docker-Netzwerks (Bridge/Overlay)',
        badgeClass: 'bg-cyan-950/40 text-cyan-400 border border-cyan-800/50',
        icon: Lock,
        iconColor: 'text-cyan-400',
      };
  }
}

function getRemoteType(ip: string): 'wan' | 'lan' | 'loopback' | 'wildcard' {
  if (!ip || ip === '0.0.0.0' || ip === '::' || ip === '*') return 'wildcard';
  if (ip.startsWith('127.') || ip === '::1' || ip === 'localhost') return 'loopback';
  if (
    ip.startsWith('10.') ||
    ip.startsWith('192.168.') ||
    /^172\.(1[6-9]|2[0-9]|3[0-1])\./.test(ip)
  ) {
    return 'lan';
  }
  return 'wan';
}

function detectAppProtocol(conn: SocketConnection): string {
  if (conn.detectedProto && conn.detectedProto !== '—') return conn.detectedProto;
  if (conn.appProtocol && conn.appProtocol !== '—') return conn.appProtocol;

  const port = conn.localPort;
  if (port === 443 || port === 8443) return 'HTTPS';
  if (port === 80 || port === 8080 || port === 8000) return 'HTTP';
  if (port === 53) return 'DNS';
  if (port === 22) return 'SSH';
  if (port === 6379) return 'Redis';
  if (port === 5432) return 'PostgreSQL';
  if (port === 3306) return 'MySQL';
  if (port === 27017) return 'MongoDB';
  if (port === 9200) return 'Elasticsearch';
  if (port === 8787) return 'Orbit Console';
  if (port === 2375 || port === 2376) return 'Docker API';
  if (port === 9090) return 'Prometheus';
  if (port === 3000) return 'Grafana/Web';
  if (port === 5672) return 'RabbitMQ';
  return conn.protocol.toUpperCase();
}

export const NetworkFlowPage: React.FC = () => {
  const [snapshot, setSnapshot] = useState<NetflowSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Filters state
  const [searchQuery, setSearchQuery] = useState('');
  const [protocolFilter, setProtocolFilter] = useState<FilterProtocol>('all');
  const [stateFilter, setStateFilter] = useState<FilterState>('all');
  const [securityFilter, setSecurityFilter] = useState<FilterSecurity>('all');
  const [selectedContainer, setSelectedContainer] = useState<string | null>(null);

  // Pagination state
  const [pageSize, setPageSize] = useState<number>(25);
  const [currentPage, setCurrentPage] = useState<number>(1);

  const loadNetflow = async (isManual = false) => {
    if (isManual) setRefreshing(true);
    else setLoading(true);

    try {
      const data = await api.getNetflow();
      setSnapshot(data);
    } catch (err) {
      console.error('Failed to load netflow:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadNetflow();

    // Subscribe to live netflow topic via WebSocket
    const unsub = wsClient.subscribe('netflow', (data: NetflowSnapshot) => {
      if (data) setSnapshot(data);
    });

    return () => unsub();
  }, []);

  // Reset page when any filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, protocolFilter, stateFilter, securityFilter, selectedContainer, pageSize]);

  const copyToClipboard = async (text: string, key: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      setTimeout(() => {
        setCopiedKey((curr) => (curr === key ? null : curr));
      }, 2000);
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
    }
  };

  // Safe data extraction
  const allConnections = useMemo(() => snapshot?.connections || [], [snapshot]);
  const containerTraffic = useMemo(() => snapshot?.containerTraffic || [], [snapshot]);
  const protocolStats = useMemo(() => snapshot?.protocolStats || [], [snapshot]);

  // Aggregate Metrics & KPI calculations
  const totalRxBps = useMemo(() => {
    if (snapshot?.totals?.totalRxBps !== undefined) return snapshot.totals.totalRxBps;
    return containerTraffic.reduce((sum, c) => sum + (c.rxBps || 0), 0);
  }, [snapshot, containerTraffic]);

  const totalTxBps = useMemo(() => {
    if (snapshot?.totals?.totalTxBps !== undefined) return snapshot.totals.totalTxBps;
    return containerTraffic.reduce((sum, c) => sum + (c.txBps || 0), 0);
  }, [snapshot, containerTraffic]);

  const totalRxBytes = useMemo(() => {
    return containerTraffic.reduce((sum, c) => sum + (c.rxBytes || 0), 0);
  }, [containerTraffic]);

  const totalTxBytes = useMemo(() => {
    return containerTraffic.reduce((sum, c) => sum + (c.txBytes || 0), 0);
  }, [containerTraffic]);

  const activeConnectionsCount = useMemo(() => {
    if (snapshot?.totals?.activeConnections !== undefined) return snapshot.totals.activeConnections;
    return allConnections.filter((c) => c.state === 'ESTABLISHED').length;
  }, [snapshot, allConnections]);

  const listeningCount = useMemo(() => {
    return allConnections.filter((c) => c.state === 'LISTEN').length;
  }, [allConnections]);

  const criticalAlertsCount = useMemo(() => {
    return allConnections.filter((c) => getSecurityLevel(c) === 'critical').length;
  }, [allConnections]);

  const cautionAlertsCount = useMemo(() => {
    return allConnections.filter((c) => getSecurityLevel(c) === 'caution').length;
  }, [allConnections]);

  const totalSecurityAlerts = useMemo(() => {
    if (snapshot?.totals?.securityAlertsCount !== undefined) return snapshot.totals.securityAlertsCount;
    return criticalAlertsCount + cautionAlertsCount;
  }, [snapshot, criticalAlertsCount, cautionAlertsCount]);

  // Top Container Traffic Ranking (Sorted by Total Throughput)
  const rankedContainers = useMemo(() => {
    const list = [...containerTraffic];
    // If empty, fall back to containers from topology nodes
    if (list.length === 0 && snapshot?.topology?.nodes) {
      const cNodes = snapshot.topology.nodes.filter((n) => n.type === 'container');
      return cNodes.map((n) => ({
        containerId: n.id,
        containerName: n.label,
        status: n.status || 'running',
        rxBps: n.rxBps?.value || 0,
        txBps: n.txBps?.value || 0,
        rxBytes: 0,
        txBytes: 0,
      }));
    }
    return list.sort((a, b) => (b.rxBps + b.txBps) - (a.rxBps + a.txBps));
  }, [containerTraffic, snapshot]);

  const maxContainerThroughput = useMemo(() => {
    if (rankedContainers.length === 0) return 1;
    const maxVal = Math.max(...rankedContainers.map((c) => c.rxBps + c.txBps));
    return maxVal > 0 ? maxVal : 1;
  }, [rankedContainers]);

  // Application Protocol Breakdown for Recharts PieChart
  const protocolChartData = useMemo(() => {
    // Categorize traffic
    const categories: Record<string, { name: string; value: number; color: string; configured: number; observed: number }> = {
      web: { name: 'Web (HTTPS/HTTP)', value: 0, color: '#3b82f6', configured: 0, observed: 0 },
      database: { name: 'Datenbanken (SQL/Redis)', value: 0, color: '#06b6d4', configured: 0, observed: 0 },
      system: { name: 'System & DNS', value: 0, color: '#f59e0b', configured: 0, observed: 0 },
      ssh: { name: 'SSH & Shell', value: 0, color: '#8b5cf6', configured: 0, observed: 0 },
      management: { name: 'Orbit & Docker', value: 0, color: '#ec4899', configured: 0, observed: 0 },
      other: { name: 'Sonstige TCP/UDP', value: 0, color: '#64748b', configured: 0, observed: 0 },
    };

    // First use protocolStats if available
    protocolStats.forEach((ps) => {
      const proto = ps.protocol.toLowerCase();
      let cat = 'other';
      if (proto.includes('http') || proto.includes('443') || proto.includes('80')) cat = 'web';
      else if (proto.includes('postgres') || proto.includes('redis') || proto.includes('mysql') || proto.includes('mongo')) cat = 'database';
      else if (proto.includes('dns') || proto.includes('53') || proto.includes('ntp')) cat = 'system';
      else if (proto.includes('ssh') || proto.includes('22')) cat = 'ssh';
      else if (proto.includes('orbit') || proto.includes('8787') || proto.includes('docker')) cat = 'management';

      categories[cat].value += ps.activeFlows || ps.observedConnectionsCount || 1;
      categories[cat].configured += ps.configuredPortsCount || 0;
      categories[cat].observed += ps.observedConnectionsCount || 0;
    });

    // Also factor in connection counts if protocolStats was empty
    if (protocolStats.length === 0 && allConnections.length > 0) {
      allConnections.forEach((conn) => {
        const app = detectAppProtocol(conn).toLowerCase();
        let cat = 'other';
        if (app.includes('http')) cat = 'web';
        else if (app.includes('redis') || app.includes('sql') || app.includes('mongo') || app.includes('elastic')) cat = 'database';
        else if (app.includes('dns')) cat = 'system';
        else if (app.includes('ssh')) cat = 'ssh';
        else if (app.includes('orbit') || app.includes('docker')) cat = 'management';

        categories[cat].value += 1;
        categories[cat].observed += 1;
      });
    }

    const data = Object.values(categories).filter((c) => c.value > 0);
    if (data.length === 0) {
      return [{ name: 'Bereit für Traffic', value: 1, color: '#282828', configured: 0, observed: 0 }];
    }
    return data;
  }, [protocolStats, allConnections]);

  // Filter Active Socket Connections
  const filteredConnections = useMemo(() => {
    return allConnections.filter((conn) => {
      // 1. Container filter
      if (selectedContainer) {
        const matchesContainer =
          (conn.containerName && conn.containerName.toLowerCase() === selectedContainer.toLowerCase()) ||
          (conn.containerId && conn.containerId.toLowerCase().includes(selectedContainer.toLowerCase()));
        if (!matchesContainer) return false;
      }

      // 2. Protocol filter
      if (protocolFilter !== 'all') {
        const p = (conn.protocol || '').toLowerCase();
        if (!p.includes(protocolFilter.toLowerCase())) return false;
      }

      // 3. State filter
      if (stateFilter !== 'all') {
        if (conn.state !== stateFilter) return false;
      }

      // 4. Security level filter
      if (securityFilter !== 'all') {
        const level = getSecurityLevel(conn);
        if (level !== securityFilter) return false;
      }

      // 5. Global text search
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase().trim();
        const local = (conn.localAddr || `${conn.localIp}:${conn.localPort}`).toLowerCase();
        const remote = (conn.remoteAddr || `${conn.remoteIp}:${conn.remotePort}`).toLowerCase();
        const cName = (conn.containerName || '').toLowerCase();
        const procName = (conn.processName || '').toLowerCase();
        const app = detectAppProtocol(conn).toLowerCase();

        return (
          local.includes(q) ||
          remote.includes(q) ||
          cName.includes(q) ||
          procName.includes(q) ||
          app.includes(q) ||
          (conn.localPort !== undefined && conn.localPort !== null && conn.localPort.toString().includes(q))
        );
      }

      return true;
    });
  }, [allConnections, selectedContainer, protocolFilter, stateFilter, securityFilter, searchQuery]);

  // Pagination slicing
  const totalPages = Math.max(1, Math.ceil(filteredConnections.length / pageSize));
  const currentPageClamped = Math.min(currentPage, totalPages);
  const startIndex = (currentPageClamped - 1) * pageSize;
  const endIndex = Math.min(startIndex + pageSize, filteredConnections.length);
  const paginatedConnections = filteredConnections.slice(startIndex, endIndex);

  const resetAllFilters = () => {
    setSearchQuery('');
    setProtocolFilter('all');
    setStateFilter('all');
    setSecurityFilter('all');
    setSelectedContainer(null);
  };

  const isFiltered =
    Boolean(searchQuery.trim()) ||
    protocolFilter !== 'all' ||
    stateFilter !== 'all' ||
    securityFilter !== 'all' ||
    selectedContainer !== null;

  if (loading || !snapshot) {
    return (
      <div className="flex flex-col items-center justify-center h-80 text-neutral-400 font-mono text-xs gap-3">
        <Activity className="w-8 h-8 text-blue-500 animate-pulse" />
        <span>Lade Netzwerktopologie und Live-Socket-Telemetrie...</span>
      </div>
    );
  }

  // Top React Flow Layout Calculations
  const rawNodes = snapshot.topology?.nodes || [];
  const rawEdges = snapshot.topology?.edges || [];

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
    ...containerNodes.map((n, i) => {
      const isSelected = selectedContainer && n.label.toLowerCase() === selectedContainer.toLowerCase();
      const isRunning = n.status === 'running';
      return {
        id: n.id,
        position: { x: 620, y: 40 + i * 75 },
        data: { label: n.label, type: n.type, status: n.status },
        style: {
          background: isSelected ? '#1e293b' : isRunning ? '#181818' : '#141414',
          color: isSelected ? '#60a5fa' : isRunning ? '#10b981' : '#737373',
          border: isSelected
            ? '2px solid #3b82f6'
            : isRunning
            ? '1px solid #059669'
            : '1px solid #282828',
          borderRadius: '6px',
          padding: '7px 14px',
          fontSize: '11px',
          fontWeight: '500',
          cursor: 'pointer',
        },
      };
    }),
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

  const handleNodeClick = (_: React.MouseEvent, node: Node) => {
    if (node.data && node.data.type === 'container') {
      const containerName = String(node.data.label);
      if (selectedContainer === containerName) {
        setSelectedContainer(null);
      } else {
        setSelectedContainer(containerName);
      }
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Top Banner Notice & Global Controls */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-lg bg-[#181818] border border-[#282828]">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-semibold text-white tracking-tight">Interaktive Netzwerktopologie</h2>
            <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-blue-950/40 text-blue-400 border border-blue-800/40 font-mono">
              Tailscale & UniFi Inspector
            </span>
          </div>
          <p className="text-xs text-neutral-400 mt-1">
            Host-System → Docker Networks → Container Verknüpfungen mit Echtzeit-Durchsatz & Port-Sicherheitsanalyse
          </p>
        </div>

        <div className="flex items-center gap-2">
          {/* Status Badge */}
          <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-md bg-[#121212] border border-[#282828] text-xs text-neutral-300">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
            <span className="font-mono text-[11px]">{snapshot.ebpfStatus || 'Telemetry Active'}</span>
          </div>

          {/* Manual Refresh Button */}
          <button
            onClick={() => loadNetflow(true)}
            disabled={refreshing}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#121212] hover:bg-[#202020] border border-[#282828] text-xs font-medium text-neutral-300 hover:text-white transition-colors disabled:opacity-50"
            title="Telemetriedaten aktualisieren"
          >
            <RefreshCw className={cn('w-3.5 h-3.5 text-neutral-400', refreshing && 'animate-spin text-blue-400')} />
            <span className="hidden sm:inline">Aktualisieren</span>
          </button>
        </div>
      </div>

      {/* Selected Container Filter Notice */}
      {selectedContainer && (
        <div className="flex items-center justify-between px-3.5 py-2 rounded-md bg-blue-950/30 border border-blue-800/40 text-xs text-blue-300">
          <div className="flex items-center gap-2">
            <Boxes className="w-4 h-4 text-blue-400" />
            <span>
              Ansicht gefiltert auf Container:{' '}
              <strong className="text-white font-mono">{selectedContainer}</strong>
            </span>
          </div>
          <button
            onClick={() => setSelectedContainer(null)}
            className="flex items-center gap-1 px-2 py-0.5 rounded bg-blue-900/40 hover:bg-blue-800/50 text-blue-200 text-[11px] transition-colors"
          >
            <X className="w-3 h-3" />
            <span>Filter aufheben</span>
          </button>
        </div>
      )}

      {/* React Flow Interactive Graph Canvas */}
      <div className="h-[460px] w-full rounded-lg overflow-hidden border border-[#282828] bg-[#121212] relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          minZoom={0.2}
          maxZoom={2}
          onNodeClick={handleNodeClick}
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

        {/* Graph Legend Overlay */}
        <div className="absolute bottom-3 left-3 flex items-center gap-3 px-2.5 py-1.5 rounded-md bg-[#181818]/90 backdrop-blur-xs border border-[#282828] text-[11px] text-neutral-400 pointer-events-none">
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-xs bg-[#181818] border border-blue-500" />
            <span>Host</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-xs bg-[#181818] border border-neutral-600" />
            <span>Netzwerk</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-xs bg-[#181818] border border-emerald-500" />
            <span>Container</span>
          </div>
          <span className="text-neutral-500">| Klick auf Container zum Filtern</span>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* LOWER SECTION: Bandwidth Rate Meters & KPI Cards                          */}
      {/* ========================================================================= */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* KPI 1: Ingress Rate (RX) */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] hover:border-[#383838] transition-colors shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-neutral-400">Ingress-Rate (RX)</span>
            <div className="w-7 h-7 rounded-md bg-emerald-950/40 border border-emerald-800/40 flex items-center justify-center text-emerald-400">
              <ArrowDown className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold font-mono text-white tracking-tight tabular-nums">
              {formatBps(totalRxBps)}
            </span>
          </div>
          <div className="mt-2 flex items-center justify-between text-[11px] text-neutral-500 font-mono">
            <span>Gesamt empfangen:</span>
            <span className="text-neutral-300 font-medium tabular-nums">{formatBytes(totalRxBytes)}</span>
          </div>
        </div>

        {/* KPI 2: Egress Rate (TX) */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] hover:border-[#383838] transition-colors shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-neutral-400">Egress-Rate (TX)</span>
            <div className="w-7 h-7 rounded-md bg-blue-950/40 border border-blue-800/40 flex items-center justify-center text-blue-400">
              <ArrowUp className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold font-mono text-white tracking-tight tabular-nums">
              {formatBps(totalTxBps)}
            </span>
          </div>
          <div className="mt-2 flex items-center justify-between text-[11px] text-neutral-500 font-mono">
            <span>Gesamt gesendet:</span>
            <span className="text-neutral-300 font-medium tabular-nums">{formatBytes(totalTxBytes)}</span>
          </div>
        </div>

        {/* KPI 3: Active Socket Connections */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] hover:border-[#383838] transition-colors shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-neutral-400">Aktive Verbindungen</span>
            <div className="w-7 h-7 rounded-md bg-cyan-950/40 border border-cyan-800/40 flex items-center justify-center text-cyan-400">
              <Activity className="w-4 h-4" />
            </div>
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold font-mono text-white tracking-tight tabular-nums">
              {activeConnectionsCount}
            </span>
            <span className="text-xs text-neutral-400 font-normal">ESTABLISHED</span>
          </div>
          <div className="mt-2 flex items-center justify-between text-[11px] text-neutral-500 font-mono">
            <span>Listening Sockets:</span>
            <span className="text-neutral-300 font-medium tabular-nums">{listeningCount} LISTEN</span>
          </div>
        </div>

        {/* KPI 4: Port Security Risk Alerts */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] hover:border-[#383838] transition-colors shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-neutral-400">Port-Sicherheitswarnungen</span>
            <div
              className={cn(
                'w-7 h-7 rounded-md flex items-center justify-center',
                totalSecurityAlerts > 0
                  ? 'bg-rose-950/40 border border-rose-800/40 text-rose-400'
                  : 'bg-emerald-950/40 border border-emerald-800/40 text-emerald-400'
              )}
            >
              {totalSecurityAlerts > 0 ? (
                <ShieldAlert className="w-4 h-4" />
              ) : (
                <ShieldCheck className="w-4 h-4" />
              )}
            </div>
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span
              className={cn(
                'text-2xl font-bold font-mono tracking-tight tabular-nums',
                criticalAlertsCount > 0
                  ? 'text-rose-400'
                  : cautionAlertsCount > 0
                  ? 'text-amber-400'
                  : 'text-emerald-400'
              )}
            >
              {totalSecurityAlerts}
            </span>
            <span className="text-xs text-neutral-400 font-normal">
              {totalSecurityAlerts === 0 ? 'Keine Risiken' : 'Warnungen'}
            </span>
          </div>
          <div className="mt-2 flex items-center justify-between text-[11px] font-mono">
            {totalSecurityAlerts > 0 ? (
              <>
                <span className="text-rose-400 font-medium">{criticalAlertsCount} Kritisch</span>
                <span className="text-amber-400 font-medium">{cautionAlertsCount} Warnung</span>
              </>
            ) : (
              <span className="text-emerald-400 font-medium">Alle Schnittstellen sicher gebunden</span>
            )}
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* LOWER SECTION: Live Traffic Breakdown by Container & Protocol             */}
      {/* ========================================================================= */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Left Card: Container Bandwidth Ranking */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                <Boxes className="w-4 h-4 text-blue-400" />
                <h3 className="text-sm font-semibold text-white tracking-tight">
                  Container Bandbreiten-Rangliste
                </h3>
              </div>
              <span className="text-[11px] font-mono text-neutral-500">
                {rankedContainers.length} Container
              </span>
            </div>
            <p className="text-xs text-neutral-400 mb-4">
              Echtzeit-Durchsatz nach Container (Ingress & Egress Durchsatz)
            </p>

            <div className="space-y-3.5">
              {rankedContainers.length === 0 ? (
                <div className="py-8 text-center text-neutral-500 text-xs">
                  Keine Container-Traffic-Daten verfügbar
                </div>
              ) : (
                rankedContainers.slice(0, 5).map((c) => {
                  const throughput = c.rxBps + c.txBps;
                  const percent = Math.min(100, Math.round((throughput / maxContainerThroughput) * 100));
                  const isSelected = selectedContainer && selectedContainer.toLowerCase() === c.containerName.toLowerCase();

                  return (
                    <div
                      key={c.containerId}
                      onClick={() =>
                        setSelectedContainer(isSelected ? null : c.containerName)
                      }
                      className={cn(
                        'group p-2.5 rounded-md border transition-all cursor-pointer',
                        isSelected
                          ? 'bg-blue-950/20 border-blue-700/50'
                          : 'bg-[#121212] border-[#242424] hover:border-[#383838] hover:bg-[#151515]'
                      )}
                    >
                      <div className="flex items-center justify-between text-xs mb-1.5">
                        <div className="flex items-center gap-2 min-w-0">
                          <StatusBadge status={c.status || 'running'} />
                          <span className="font-medium text-white truncate group-hover:text-blue-400 transition-colors">
                            {c.containerName}
                          </span>
                          <span className="text-[10px] font-mono text-neutral-500 hidden sm:inline">
                            {c.containerId.slice(0, 10)}
                          </span>
                        </div>

                        <div className="flex items-center gap-3 shrink-0 font-mono text-[11px]">
                          <span className="text-emerald-400 font-medium tabular-nums flex items-center gap-0.5">
                            <ArrowDown className="w-3 h-3" />
                            {formatBps(c.rxBps)}
                          </span>
                          <span className="text-blue-400 font-medium tabular-nums flex items-center gap-0.5">
                            <ArrowUp className="w-3 h-3" />
                            {formatBps(c.txBps)}
                          </span>
                        </div>
                      </div>

                      {/* Proportional Bandwidth Bar */}
                      <div className="w-full bg-[#1e1e1e] h-1.5 rounded-full overflow-hidden">
                        <div
                          className="h-full rounded-full bg-linear-to-r from-emerald-500 to-blue-500 transition-all duration-300"
                          style={{ width: `${Math.max(percent, 4)}%` }}
                        />
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          <div className="mt-4 pt-3 border-t border-[#242424] flex items-center justify-between text-[11px] text-neutral-500 font-mono">
            <span>Klick auf Zeile filtert Sockets</span>
            <span>Gesamtdurchsatz: {formatBps(totalRxBps + totalTxBps)}</span>
          </div>
        </div>

        {/* Right Card: Application Protocol Breakdown (Recharts Donut) */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                <Globe className="w-4 h-4 text-purple-400" />
                <h3 className="text-sm font-semibold text-white tracking-tight">
                  Protokoll- & Anwendungsverteilung
                </h3>
              </div>
              <span className="text-[11px] font-mono text-neutral-500">
                {protocolStats.length || protocolChartData.length} Kategorien
              </span>
            </div>
            <p className="text-xs text-neutral-400 mb-3">
              Verteilung aktiver Socket-Flows & App-Klassifikation
            </p>

            <div className="flex flex-col sm:flex-row items-center gap-4">
              {/* Recharts PieChart (Donut) */}
              <div className="h-44 w-44 shrink-0 relative flex items-center justify-center">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={protocolChartData}
                      dataKey="value"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      innerRadius={44}
                      outerRadius={70}
                      paddingAngle={3}
                      stroke="#181818"
                      strokeWidth={2}
                    >
                      {protocolChartData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        backgroundColor: '#141414',
                        borderColor: '#282828',
                        borderRadius: '0.375rem',
                        fontSize: '11px',
                        color: '#f5f5f5',
                        boxShadow: '0 4px 12px rgba(0,0,0,0.5)',
                      }}
                      formatter={(val: any) => [`${val} Flows`, 'Aktiv']}
                    />
                  </PieChart>
                </ResponsiveContainer>

                {/* Donut Center Label */}
                <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                  <span className="text-base font-bold font-mono text-white tabular-nums">
                    {allConnections.length}
                  </span>
                  <span className="text-[9px] uppercase tracking-wider text-neutral-500">Flows</span>
                </div>
              </div>

              {/* Protocol Breakdown Legend List */}
              <div className="flex-1 w-full space-y-2">
                {protocolChartData.map((p) => (
                  <div
                    key={p.name}
                    className="flex items-center justify-between p-1.5 rounded-md bg-[#121212] border border-[#242424] text-xs font-mono"
                  >
                    <div className="flex items-center gap-2 truncate">
                      <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: p.color }} />
                      <span className="text-neutral-300 truncate">{p.name}</span>
                    </div>
                    <div className="flex items-center gap-2 shrink-0 text-[11px] text-neutral-400 tabular-nums">
                      <span className="font-medium text-white">{p.value} Flows</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="mt-4 pt-3 border-t border-[#242424] flex items-center justify-between text-[11px] text-neutral-500 font-mono">
            <span>DPI / Socket-Inspektion</span>
            <span>{listeningCount} Listening Ports</span>
          </div>
        </div>
      </div>

      {/* ========================================================================= */}
      {/* LOWER SECTION: Active Socket Connections Table                            */}
      {/* ========================================================================= */}
      <div className="p-4 rounded-lg bg-[#181818] border border-[#282828]">
        {/* Table Header & Title */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 mb-4">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-base font-semibold text-white tracking-tight">
                Aktive Socket-Verbindungen
              </h3>
              <span className="px-2 py-0.5 rounded text-xs font-mono font-medium bg-[#121212] text-neutral-300 border border-[#282828]">
                {filteredConnections.length} erfasst
              </span>
            </div>
            <p className="text-xs text-neutral-400 mt-0.5">
              Verbindungsstatus, Container-Zuordnung und automatisierte Port-Sicherheitsbewertung
            </p>
          </div>

          {/* Quick Action: Reset Filters */}
          {isFiltered && (
            <button
              onClick={resetAllFilters}
              className="self-start md:self-auto flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#202020] hover:bg-[#282828] text-xs font-medium text-neutral-300 hover:text-white transition-colors border border-[#333333]"
            >
              <X className="w-3.5 h-3.5" />
              <span>Filter zurücksetzen</span>
            </button>
          )}
        </div>

        {/* Filter & Search Bar */}
        <div className="flex flex-col lg:flex-row gap-3 mb-4">
          {/* Global Search Input */}
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-neutral-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Suchen nach Container, Prozess, Port oder IP-Adresse..."
              className="w-full pl-9 pr-8 py-1.5 rounded-md bg-[#121212] border border-[#282828] text-xs text-white placeholder-neutral-500 focus:outline-hidden focus:border-blue-500 transition-colors"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-white"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          {/* Filter Pills Controls */}
          <div className="flex flex-wrap items-center gap-2">
            {/* Protocol Selector */}
            <div className="flex items-center bg-[#121212] border border-[#282828] p-0.5 rounded-md text-xs">
              {(['all', 'tcp', 'udp'] as const).map((p) => (
                <button
                  key={p}
                  onClick={() => setProtocolFilter(p)}
                  className={cn(
                    'px-2.5 py-1 rounded text-xs font-mono font-medium transition-colors',
                    protocolFilter === p
                      ? 'bg-blue-600 text-white shadow-xs'
                      : 'text-neutral-400 hover:text-white'
                  )}
                >
                  {p.toUpperCase()}
                </button>
              ))}
            </div>

            {/* State Selector */}
            <div className="flex items-center bg-[#121212] border border-[#282828] p-0.5 rounded-md text-xs">
              {(['all', 'ESTABLISHED', 'LISTEN', 'TIME_WAIT'] as const).map((st) => (
                <button
                  key={st}
                  onClick={() => setStateFilter(st)}
                  className={cn(
                    'px-2 py-1 rounded text-[11px] font-medium transition-colors',
                    stateFilter === st
                      ? 'bg-white/10 text-white shadow-xs'
                      : 'text-neutral-400 hover:text-white'
                  )}
                >
                  {st === 'all' ? 'Alle Status' : st}
                </button>
              ))}
            </div>

            {/* Security Risk Filter Selector */}
            <div className="flex items-center bg-[#121212] border border-[#282828] p-0.5 rounded-md text-xs">
              {(
                [
                  { id: 'all', label: 'Alle Risiken' },
                  { id: 'critical', label: '🔴 Kritisch' },
                  { id: 'caution', label: '⚠️ Warnung' },
                  { id: 'safe', label: '🛡️ Sicher' },
                  { id: 'internal', label: '🔒 Intern' },
                ] as const
              ).map((sec) => (
                <button
                  key={sec.id}
                  onClick={() => setSecurityFilter(sec.id)}
                  className={cn(
                    'px-2 py-1 rounded text-[11px] font-medium transition-colors',
                    securityFilter === sec.id
                      ? 'bg-white/10 text-white shadow-xs'
                      : 'text-neutral-400 hover:text-white'
                  )}
                >
                  {sec.label}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Table Container */}
        <div className="overflow-x-auto rounded-md border border-[#242424] bg-[#121212]">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold bg-[#151515]">
                <th className="py-2.5 px-3 font-semibold">Sicherheitsstufe</th>
                <th className="py-2.5 px-3 font-semibold">Container & Prozess</th>
                <th className="py-2.5 px-3 font-semibold">Lokale Adresse : Port</th>
                <th className="py-2.5 px-3 font-semibold">Remote Adresse : Port</th>
                <th className="py-2.5 px-3 font-semibold">Protokoll</th>
                <th className="py-2.5 px-3 font-semibold">Status</th>
                <th className="py-2.5 px-3 font-semibold text-right">App / Klassifikation</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1f1f1f]">
              {paginatedConnections.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-12 text-center">
                    <div className="flex flex-col items-center justify-center gap-2 text-neutral-400">
                      <Filter className="w-6 h-6 text-neutral-500" />
                      <span className="font-medium text-white text-sm">
                        Keine Verbindungen gefunden
                      </span>
                      <p className="text-xs text-neutral-500 max-w-sm text-center">
                        Es wurden keine Sockets für die aktuellen Filter- und Suchkriterien gefunden.
                      </p>
                      {isFiltered && (
                        <button
                          onClick={resetAllFilters}
                          className="mt-2 px-3 py-1 rounded bg-[#202020] hover:bg-[#282828] text-xs font-medium text-neutral-200 border border-[#333333] transition-colors"
                        >
                          Alle Filter zurücksetzen
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ) : (
                paginatedConnections.map((conn, idx) => {
                  const secLevel = getSecurityLevel(conn);
                  const secMeta = getSecurityMeta(secLevel, conn.securityDesc);
                  const SecIcon = secMeta.icon;

                  const localAddress = conn.localAddr || `${conn.localIp}:${conn.localPort}`;
                  const remoteAddress = conn.remoteAddr || `${conn.remoteIp}:${conn.remotePort}`;
                  const appName = detectAppProtocol(conn);
                  const rType = getRemoteType(conn.remoteIp);

                  const localCopyKey = `loc-${conn.containerId || ''}-${localAddress}-${idx}`;
                  const remoteCopyKey = `rem-${conn.containerId || ''}-${remoteAddress}-${idx}`;

                  return (
                    <tr
                      key={`${conn.localAddr}-${conn.remoteAddr}-${idx}`}
                      className="hover:bg-white/[0.02] transition-colors group"
                    >
                      {/* Column 1: Port Security Indicator */}
                      <td className="py-2.5 px-3">
                        <span
                          className={cn(
                            'inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-medium whitespace-nowrap',
                            secMeta.badgeClass
                          )}
                          title={secMeta.desc}
                        >
                          <SecIcon className={cn('w-3.5 h-3.5 shrink-0', secMeta.iconColor)} />
                          <span>{secMeta.label}</span>
                        </span>
                      </td>

                      {/* Column 2: Container Attribution & Process */}
                      <td className="py-2.5 px-3">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() =>
                              conn.containerName &&
                              setSelectedContainer(
                                selectedContainer === conn.containerName ? null : conn.containerName
                              )
                            }
                            className="font-medium text-white hover:text-blue-400 transition-colors flex items-center gap-1.5"
                            title="Nach diesem Container filtern"
                          >
                            <Boxes className="w-3.5 h-3.5 text-neutral-400" />
                            <span>{conn.containerName || '—'}</span>
                          </button>
                        </div>
                        {conn.processName && (
                          <div className="text-[10px] font-mono text-neutral-400 mt-0.5 truncate max-w-[160px]" title={conn.processName}>
                            proc: {conn.processName}
                          </div>
                        )}
                      </td>

                      {/* Column 3: Local Address & Port */}
                      <td className="py-2.5 px-3">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-neutral-200 tabular-nums text-xs">
                            {localAddress}
                          </span>
                          <button
                            onClick={(e) => copyToClipboard(localAddress, localCopyKey, e)}
                            className="opacity-0 group-hover:opacity-100 text-neutral-400 hover:text-white transition-opacity p-0.5 rounded"
                            title="Lokale Adresse kopieren"
                          >
                            {copiedKey === localCopyKey ? (
                              <Check className="w-3 h-3 text-emerald-400" />
                            ) : (
                              <Copy className="w-3 h-3" />
                            )}
                          </button>
                        </div>
                      </td>

                      {/* Column 4: Remote Address & Port */}
                      <td className="py-2.5 px-3">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-neutral-200 tabular-nums text-xs">
                            {remoteAddress}
                          </span>
                          {rType === 'lan' && (
                            <span className="px-1 py-0.2 rounded text-[9px] font-mono bg-blue-950/40 text-blue-400 border border-blue-900/40">
                              LAN
                            </span>
                          )}
                          {rType === 'wan' && (
                            <span className="px-1 py-0.2 rounded text-[9px] font-mono bg-purple-950/40 text-purple-400 border border-purple-900/40">
                              WAN
                            </span>
                          )}
                          {remoteAddress !== '0.0.0.0:*' && remoteAddress !== '*:*' && (
                            <button
                              onClick={(e) => copyToClipboard(remoteAddress, remoteCopyKey, e)}
                              className="opacity-0 group-hover:opacity-100 text-neutral-400 hover:text-white transition-opacity p-0.5 rounded"
                              title="Remote Adresse kopieren"
                            >
                              {copiedKey === remoteCopyKey ? (
                                <Check className="w-3 h-3 text-emerald-400" />
                              ) : (
                                <Copy className="w-3 h-3" />
                              )}
                            </button>
                          )}
                        </div>
                      </td>

                      {/* Column 5: Protocol */}
                      <td className="py-2.5 px-3">
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-mono font-medium bg-[#1e1e1e] text-neutral-300 border border-[#2c2c2c]">
                          {conn.protocol.toUpperCase()}
                        </span>
                      </td>

                      {/* Column 6: Connection State */}
                      <td className="py-2.5 px-3">
                        <span
                          className={cn(
                            'inline-block px-1.5 py-0.5 rounded text-[10px] font-medium whitespace-nowrap',
                            conn.state === 'ESTABLISHED'
                              ? 'bg-emerald-950/40 text-emerald-400 border border-emerald-800/40'
                              : conn.state === 'LISTEN'
                              ? 'bg-neutral-800 text-neutral-300 border border-neutral-700'
                              : conn.state === 'TIME_WAIT'
                              ? 'bg-amber-950/40 text-amber-400 border border-amber-800/40'
                              : 'bg-zinc-800 text-zinc-400 border border-zinc-700'
                          )}
                        >
                          {conn.state}
                        </span>
                      </td>

                      {/* Column 7: App Classification */}
                      <td className="py-2.5 px-3 text-right">
                        <div className="inline-flex flex-col items-end">
                          <span className="text-white font-medium font-mono text-xs">
                            {appName}
                          </span>
                          <span className="text-[10px] text-neutral-500">
                            {conn.classification === 'ebpf_observed' ? 'eBPF Trace' : 'Port-basiert'}
                          </span>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination Controls */}
        <div className="mt-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs text-neutral-400 font-mono">
          <div className="flex items-center gap-3">
            <span>
              Zeige <strong className="text-white tabular-nums">{filteredConnections.length > 0 ? startIndex + 1 : 0}</strong> bis{' '}
              <strong className="text-white tabular-nums">{endIndex}</strong> von{' '}
              <strong className="text-white tabular-nums">{filteredConnections.length}</strong> Verbindungen
            </span>

            {/* Page Size Selector */}
            <div className="flex items-center gap-1.5 ml-2">
              <span className="text-neutral-500">Pro Seite:</span>
              <div className="flex items-center bg-[#121212] border border-[#282828] p-0.5 rounded">
                {([10, 25, 50, 100] as const).map((sz) => (
                  <button
                    key={sz}
                    onClick={() => setPageSize(sz)}
                    className={cn(
                      'px-1.5 py-0.5 rounded text-[11px] font-mono transition-colors',
                      pageSize === sz
                        ? 'bg-blue-600 text-white font-bold'
                        : 'text-neutral-400 hover:text-white'
                    )}
                  >
                    {sz}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Navigation Buttons */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => setCurrentPage(1)}
              disabled={currentPageClamped <= 1}
              className="p-1 rounded bg-[#121212] hover:bg-[#202020] border border-[#282828] text-neutral-300 disabled:opacity-30 disabled:pointer-events-none transition-colors"
              title="Erste Seite"
            >
              <ChevronsLeft className="w-4 h-4" />
            </button>
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPageClamped <= 1}
              className="p-1 rounded bg-[#121212] hover:bg-[#202020] border border-[#282828] text-neutral-300 disabled:opacity-30 disabled:pointer-events-none transition-colors"
              title="Vorherige Seite"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>

            <span className="px-2.5 py-1 rounded bg-[#121212] border border-[#282828] text-neutral-200 text-xs tabular-nums">
              Seite {currentPageClamped} / {totalPages}
            </span>

            <button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPageClamped >= totalPages}
              className="p-1 rounded bg-[#121212] hover:bg-[#202020] border border-[#282828] text-neutral-300 disabled:opacity-30 disabled:pointer-events-none transition-colors"
              title="Nächste Seite"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
            <button
              onClick={() => setCurrentPage(totalPages)}
              disabled={currentPageClamped >= totalPages}
              className="p-1 rounded bg-[#121212] hover:bg-[#202020] border border-[#282828] text-neutral-300 disabled:opacity-30 disabled:pointer-events-none transition-colors"
              title="Letzte Seite"
            >
              <ChevronsRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
