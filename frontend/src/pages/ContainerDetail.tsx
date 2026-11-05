import React, { useState, useEffect, useRef } from 'react';
import {
  ArrowLeft,
  Play,
  Square,
  RotateCw,
  Terminal,
  Activity,
  Network,
  Code,
  Info,
  Download,
  Search,
  Copy,
  Check,
} from 'lucide-react';
import { Container, ContainerStats, AuthStatus } from '../types';
import { StatusBadge } from '../components/common/StatusBadge';
import { formatBytes, formatBps, formatPercent, formatUptime, formatDate } from '../lib/utils';
import { api } from '../lib/api';
import { wsClient } from '../lib/ws';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface ContainerDetailProps {
  containerId: string;
  authStatus?: AuthStatus;
  onBack: () => void;
  onRefresh: () => void;
}

export const ContainerDetail: React.FC<ContainerDetailProps> = ({
  containerId,
  authStatus,
  onBack,
  onRefresh,
}) => {
  const [data, setData] = useState<{ container: Container; raw?: any; inspect?: any } | null>(null);
  const [stats, setStats] = useState<ContainerStats | null>(null);
  const [history, setHistory] = useState<ContainerStats[]>([]);
  const [logs, setLogs] = useState<string>('');
  const [logSearch, setLogSearch] = useState('');
  const [activeTab, setActiveTab] = useState<'overview' | 'performance' | 'network' | 'logs' | 'inspect'>('overview');
  const [loading, setLoading] = useState(true);
  const [copiedInspect, setCopiedInspect] = useState(false);
  const logEndRef = useRef<HTMLDivElement>(null);

  const actionsEnabled = authStatus?.features.enableActions ?? false;

  const loadData = async () => {
    try {
      const res = await api.getContainer(containerId);
      setData(res);
      if (res.container.stats) {
        setStats(res.container.stats);
      }
      const hist = await api.getContainerStatsHistory(containerId);
      setHistory(hist);
    } catch (err: any) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadLogs = async () => {
    try {
      const text = await api.getContainerLogs(containerId, 500, true);
      setLogs(text);
    } catch (err: any) {
      setLogs(`Fehler beim Laden der Logs: ${err.message}`);
    }
  };

  useEffect(() => {
    loadData();
    loadLogs();

    // Subscribe to live stats for this specific container
    const unsub = wsClient.subscribe(`container:${containerId}`, (newStat: ContainerStats) => {
      setStats(newStat);
      setHistory((prev) => [...prev.slice(-59), newStat]);
    });

    return () => unsub();
  }, [containerId]);

  if (loading || !data) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 font-mono text-xs">
        Lade Container-Details...
      </div>
    );
  }

  const c = data.container;
  const isRunning = c.state === 'running';

  const chartData = history.map((st) => ({
    time: new Date(st.ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
    cpu: st.cpuPercent.value ?? 0,
    ramMb: st.memUsage.value ? Math.round(st.memUsage.value / (1024 * 1024)) : 0,
    rxBps: st.netRxBps.value ?? 0,
    txBps: st.netTxBps.value ?? 0,
  }));

  const filteredLogs = logs
    .split('\n')
    .filter((line) => !logSearch || line.toLowerCase().includes(logSearch.toLowerCase()))
    .join('\n');

  const downloadLogs = () => {
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${c.name}-logs.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const copyInspect = () => {
    navigator.clipboard.writeText(JSON.stringify(data.inspect || data.raw, null, 2));
    setCopiedInspect(true);
    setTimeout(() => setCopiedInspect(false), 2000);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Top Bar with Container Info & Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="p-1.5 rounded-md bg-[#181818] border border-[#282828] text-neutral-400 hover:text-white hover:bg-[#222222] transition-colors"
            title="Zurück zur Liste"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <div>
            <div className="flex items-center gap-2.5">
              <h2 className="text-2xl font-semibold text-white tracking-tight">{c.name}</h2>
              <StatusBadge status={c.state} />
              {c.health && c.health !== 'none' && <StatusBadge status={c.health} />}
            </div>
            <p className="text-xs text-neutral-400 mt-1">
              ID: <span className="font-mono text-neutral-300">{c.shortId}</span> · Image: <span className="font-mono text-neutral-300">{c.image}</span>
            </p>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2">
          {isRunning ? (
            <>
              <button
                onClick={async () => {
                  await api.restartContainer(c.id);
                  onRefresh();
                  loadData();
                }}
                disabled={!actionsEnabled}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-[#181818] border border-[#282828] hover:bg-[#222222] disabled:opacity-30 text-neutral-300 rounded-md text-xs font-medium transition-colors"
              >
                <RotateCw className="w-3.5 h-3.5" />
                <span>Neustarten</span>
              </button>
              <button
                onClick={async () => {
                  await api.stopContainer(c.id);
                  onRefresh();
                  loadData();
                }}
                disabled={!actionsEnabled}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-30 text-neutral-200 rounded-md text-xs font-medium transition-colors"
              >
                <Square className="w-3.5 h-3.5 fill-current" />
                <span>Stoppen</span>
              </button>
            </>
          ) : (
            <button
              onClick={async () => {
                await api.startContainer(c.id);
                onRefresh();
                loadData();
              }}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-30 text-white rounded-md text-xs font-medium transition-colors"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Starten</span>
            </button>
          )}
        </div>
      </div>

      {/* Detail Navigation Tabs */}
      <div className="flex items-center gap-1 border-b border-[#242424] pb-2">
        {[
          { id: 'overview', label: 'Overview', icon: Info },
          { id: 'performance', label: 'Performance', icon: Activity },
          { id: 'network', label: 'Network', icon: Network },
          { id: 'logs', label: 'Logs', icon: Terminal },
          { id: 'inspect', label: 'Inspect', icon: Code },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                isActive
                  ? 'bg-white/10 text-white'
                  : 'text-neutral-400 hover:text-white hover:bg-white/[0.04]'
              }`}
            >
              <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-white' : 'text-neutral-500'}`} />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] space-y-3">
            <h3 className="text-xs font-semibold text-neutral-300 uppercase tracking-wider font-sans">Container Details</h3>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Status</span>
                <span className="font-semibold text-white">{c.state}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Erstellt am</span>
                <span className="font-mono text-neutral-300">{formatDate(c.created)}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Zuletzt gestartet</span>
                <span className="font-mono text-neutral-300">{formatDate(c.startedAt)}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Laufzeit</span>
                <span className="font-mono text-emerald-400 font-medium">
                  {c.startedAt ? formatUptime(Math.floor((Date.now() - new Date(c.startedAt).getTime()) / 1000)) : '—'}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Neustarts</span>
                <span className="font-mono text-neutral-300">{c.restartCount}</span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Restart Policy</span>
                <span className="font-mono text-neutral-300">{c.restartPolicy}</span>
              </div>
            </div>
          </div>

          <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] space-y-3">
            <h3 className="text-xs font-semibold text-neutral-300 uppercase tracking-wider font-sans">Resource Limits</h3>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">CPU Limit</span>
                <span className="font-mono text-neutral-300">
                  {c.limits.cpus > 0 ? `${c.limits.cpus} Kerne` : 'Unbegrenzt'}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">RAM Limit</span>
                <span className="font-mono text-neutral-300">
                  {c.limits.memoryBytes > 0 ? formatBytes(c.limits.memoryBytes) : 'Unbegrenzt (Host-RAM)'}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">PID Limit</span>
                <span className="font-mono text-neutral-300">
                  {c.limits.pidsLimit > 0 ? c.limits.pidsLimit : 'Unbegrenzt'}
                </span>
              </div>
              <div className="flex justify-between py-1.5 border-b border-[#242424]">
                <span className="text-neutral-400">Privilegiert</span>
                <span className="text-neutral-300">{c.privileged ? 'Ja' : 'Nein'}</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'performance' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 rounded-lg bg-[#181818] border border-[#282828]">
            <h3 className="text-xs font-semibold text-white uppercase tracking-wider mb-3">
              CPU Auslastung ({stats?.cpuPercent.value !== null && stats?.cpuPercent.value !== undefined ? formatPercent(stats.cpuPercent.value) : '—'})
            </h3>
            <div className="h-56">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <XAxis dataKey="time" stroke="#525252" fontSize={10} />
                  <YAxis stroke="#525252" fontSize={10} unit="%" />
                  <Tooltip contentStyle={{ backgroundColor: '#141414', borderColor: '#282828', fontSize: '11px', color: '#f5f5f5' }} />
                  <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.15} isAnimationActive={false} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="p-4 rounded-lg bg-[#181818] border border-[#282828]">
            <h3 className="text-xs font-semibold text-white uppercase tracking-wider mb-3">
              RAM Verbrauch ({formatBytes(stats?.memUsage.value)})
            </h3>
            <div className="h-56">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <XAxis dataKey="time" stroke="#525252" fontSize={10} />
                  <YAxis stroke="#525252" fontSize={10} unit="MB" />
                  <Tooltip contentStyle={{ backgroundColor: '#141414', borderColor: '#282828', fontSize: '11px', color: '#f5f5f5' }} />
                  <Area type="monotone" dataKey="ramMb" stroke="#a855f7" fill="#a855f7" fillOpacity={0.15} isAnimationActive={false} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'network' && (
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] space-y-4">
          <h3 className="text-xs font-semibold text-white uppercase tracking-wider">Netzwerk-Konfiguration</h3>
          <div className="space-y-2">
            {c.networks.map((net, i) => (
              <div key={i} className="p-3 bg-[#121212] rounded-md border border-[#242424] flex justify-between items-center text-xs">
                <div>
                  <span className="font-semibold text-white block">{net.name}</span>
                  <span className="text-neutral-500 font-mono text-[10px]">Netzwerk-ID: {net.networkId.slice(0, 12)}</span>
                </div>
                <div className="text-right font-mono text-neutral-300">
                  <span>IP: {net.ipAddress || '—'}</span>
                  {net.gateway && <span className="text-neutral-500 block text-[10px]">Gateway: {net.gateway}</span>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'logs' && (
        <div className="rounded-lg bg-[#121212] border border-[#282828] overflow-hidden flex flex-col h-[520px]">
          <div className="p-3 bg-[#181818] border-b border-[#282828] flex items-center justify-between gap-3">
            <div className="relative flex-1 max-w-sm">
              <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
              <input
                type="text"
                value={logSearch}
                onChange={(e) => setLogSearch(e.target.value)}
                placeholder="In Logs suchen..."
                className="w-full pl-8 pr-3 py-1.5 bg-[#121212] border border-[#282828] rounded-md text-xs text-neutral-200 placeholder-neutral-500 focus:outline-none focus:border-neutral-400 font-mono"
              />
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={loadLogs}
                className="px-3 py-1.5 rounded-md bg-[#181818] border border-[#282828] hover:bg-[#222222] text-xs text-neutral-300 transition-colors"
              >
                Aktualisieren
              </button>
              <button
                onClick={downloadLogs}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#181818] border border-[#282828] hover:bg-[#222222] text-xs text-neutral-300 transition-colors"
              >
                <Download className="w-3.5 h-3.5" />
                <span>Exportieren</span>
              </button>
            </div>
          </div>
          <div className="flex-1 p-4 font-mono text-xs text-neutral-300 overflow-y-auto whitespace-pre-wrap leading-relaxed select-text">
            {filteredLogs || 'Keine Logs vorhanden.'}
            <div ref={logEndRef} />
          </div>
        </div>
      )}

      {activeTab === 'inspect' && (
        <div className="rounded-lg bg-[#121212] border border-[#282828] overflow-hidden">
          <div className="p-3 bg-[#181818] border-b border-[#282828] flex items-center justify-between">
            <span className="text-xs font-mono text-neutral-400">Docker Inspect JSON</span>
            <button
              onClick={copyInspect}
              className="flex items-center gap-1.5 px-3 py-1 rounded-md bg-[#181818] border border-[#282828] hover:bg-[#222222] text-xs text-neutral-300 transition-colors"
            >
              {copiedInspect ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedInspect ? 'Kopiert' : 'Kopieren'}</span>
            </button>
          </div>
          <pre className="p-4 font-mono text-xs text-neutral-300 overflow-x-auto max-h-[500px] leading-relaxed">
            {JSON.stringify(data.inspect || data.raw, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
};

