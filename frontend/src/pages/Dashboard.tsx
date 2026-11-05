import React, { useState, useEffect } from 'react';
import {
  Cpu,
  HardDrive,
  Activity,
  ArrowDown,
  ArrowUp,
  Server,
  Boxes,
  Layers,
  Globe,
  Database,
  Thermometer,
  Zap,
} from 'lucide-react';
import { MetricCard } from '../components/common/MetricCard';
import { StatusBadge } from '../components/common/StatusBadge';
import { Container, HostMetrics, GPUTelemetry, SystemInfoResponse } from '../types';
import { formatBytes, formatBps, formatPercent, formatUptime } from '../lib/utils';
import { wsClient } from '../lib/ws';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface DashboardProps {
  initialSystemInfo?: SystemInfoResponse;
  initialMetrics?: HostMetrics;
  initialContainers?: Container[];
  initialGPU?: GPUTelemetry;
  onSelectContainer: (id: string) => void;
}

export const Dashboard: React.FC<DashboardProps> = ({
  initialSystemInfo,
  initialMetrics,
  initialContainers = [],
  initialGPU,
  onSelectContainer,
}) => {
  const [metrics, setMetrics] = useState<HostMetrics | undefined>(initialMetrics);
  const [containers, setContainers] = useState<Container[]>(initialContainers);
  const [gpu, setGpu] = useState<GPUTelemetry | undefined>(initialGPU);
  const [sysInfo, setSysInfo] = useState<SystemInfoResponse | undefined>(initialSystemInfo);

  // Live Chart rolling points (last 30 samples)
  const [chartData, setChartData] = useState<Array<{ time: string; cpu: number; ram: number; rx: number; tx: number }>>([]);

  useEffect(() => {
    if (initialMetrics) setMetrics(initialMetrics);
    if (initialContainers) setContainers(initialContainers);
    if (initialGPU) setGpu(initialGPU);
    if (initialSystemInfo) setSysInfo(initialSystemInfo);
  }, [initialMetrics, initialContainers, initialGPU, initialSystemInfo]);

  useEffect(() => {
    // Subscribe to live WebSocket topics
    const unsubHost = wsClient.subscribe('host', (data: HostMetrics) => {
      setMetrics(data);
      setChartData((prev) => {
        const now = new Date(data.timestamp || Date.now()).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        const cpuVal = data.cpu.totalPercent.value ?? 0;
        const ramVal = data.memory.usedPercent.value ?? 0;
        const rxVal = data.network.rxBps.value ?? 0;
        const txVal = data.network.txBps.value ?? 0;

        const next = [...prev, { time: now, cpu: cpuVal, ram: ramVal, rx: rxVal, tx: txVal }];
        return next.length > 30 ? next.slice(1) : next;
      });
    });

    const unsubGpu = wsClient.subscribe('gpu', (data: GPUTelemetry) => {
      setGpu(data);
    });

    const unsubContainers = wsClient.subscribe('containers', (stat: any) => {
      if (stat && stat.id) {
        setContainers((prev) =>
          prev.map((c) => (c.id === stat.id ? { ...c, stats: stat } : c))
        );
      }
    });

    return () => {
      unsubHost();
      unsubGpu();
      unsubContainers();
    };
  }, []);

  const counts = sysInfo?.inventoryCounts || {
    total: containers.length,
    running: containers.filter((c) => c.state === 'running').length,
    stopped: containers.filter((c) => c.state !== 'running').length,
    unhealthy: containers.filter((c) => c.health === 'unhealthy').length,
    healthy: containers.filter((c) => c.health === 'healthy').length,
  };

  const primaryGPU = gpu?.devices && gpu.devices.length > 0 ? gpu.devices[0] : null;

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Page Header (Tailscale style) */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Dashboard</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Zentrale Host-Telemetrie und Docker-Ressourcen im Sekundentakt.
          </p>
        </div>
      </div>

      {/* Upper Information Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* CPU Card */}
        <MetricCard
          title="CPU Gesamt"
          gauge={metrics?.cpu.totalPercent}
          formattedValue={formatPercent(metrics?.cpu.totalPercent.value)}
          progressPercent={metrics?.cpu.totalPercent.value ?? undefined}
          subtitle={`${metrics?.cpu.cores ?? '—'} Kerne · ${metrics?.cpu.threads ?? '—'} Threads${
            metrics?.cpu.freqMHz.value ? ` · ${metrics.cpu.freqMHz.value} MHz` : ''
          }`}
        />

        {/* RAM Card */}
        <MetricCard
          title="Arbeitsspeicher"
          gauge={metrics?.memory.usedPercent}
          formattedValue={formatPercent(metrics?.memory.usedPercent.value)}
          progressPercent={metrics?.memory.usedPercent.value ?? undefined}
          subtitle={`${formatBytes(metrics?.memory.usedBytes)} von ${formatBytes(metrics?.memory.totalBytes)}`}
        />

        {/* Network RX/TX Card */}
        <MetricCard
          title="Netzwerk-Durchsatz"
          formattedValue={formatBps((metrics?.network.rxBps.value ?? 0) + (metrics?.network.txBps.value ?? 0))}
          subtitle={`↓ ${formatBps(metrics?.network.rxBps.value)} · ↑ ${formatBps(metrics?.network.txBps.value)}`}
        />

        {/* Storage Card */}
        <MetricCard
          title="Festplattenspeicher"
          gauge={metrics?.storage.usedPercent}
          formattedValue={formatPercent(metrics?.storage.usedPercent.value)}
          progressPercent={metrics?.storage.usedPercent.value ?? undefined}
          subtitle={`${formatBytes(metrics?.storage.usedBytes)} von ${formatBytes(metrics?.storage.totalBytes)}`}
        />
      </div>

      {/* Secondary Metrics & Docker Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Load Average */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <span className="text-xs font-medium text-neutral-400 block mb-2">Load Average</span>
          <div className="flex items-center justify-between text-white mt-3">
            <div>
              <span className="text-[11px] text-neutral-500 block font-normal">1 min</span>
              <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg1.value?.toFixed(2) ?? '—'}</span>
            </div>
            <div>
              <span className="text-[11px] text-neutral-500 block font-normal">5 min</span>
              <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg5.value?.toFixed(2) ?? '—'}</span>
            </div>
            <div>
              <span className="text-[11px] text-neutral-500 block font-normal">15 min</span>
              <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg15.value?.toFixed(2) ?? '—'}</span>
            </div>
          </div>
        </div>

        {/* Temperature & Host Power */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <span className="text-xs font-medium text-neutral-400 block mb-2">Systemtemperatur</span>
          <div className="flex items-baseline gap-2 mt-2">
            <span className="text-2xl font-semibold tracking-tight text-white tabular-nums font-mono">
              {metrics?.cpu.tempC.value ? `${metrics.cpu.tempC.value} °C` : '—'}
            </span>
            <span className="text-xs text-neutral-500 font-normal">CPU</span>
          </div>
          <p className="text-xs text-neutral-500 mt-2 font-normal font-mono">
            Uptime: {formatUptime(metrics?.system.uptimeSeconds.value)}
          </p>
        </div>

        {/* GPU Telemetry Preview */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <span className="text-xs font-medium text-neutral-400 block mb-2">GPU Telemetrie</span>
          {primaryGPU ? (
            <div>
              <div className="flex items-baseline gap-2 mt-2">
                <span className="text-xl font-semibold tracking-tight text-white tabular-nums font-mono">
                  {formatPercent(primaryGPU.utilizationPercent.value)}
                </span>
                <span className="text-xs text-neutral-500 font-normal">Auslastung</span>
              </div>
              <p className="text-xs text-neutral-500 mt-2 font-normal truncate">
                {primaryGPU.model} {primaryGPU.temperatureCelsius.value ? `(${primaryGPU.temperatureCelsius.value}°C)` : ''}
              </p>
            </div>
          ) : (
            <div className="py-2.5">
              <span className="text-xs text-neutral-500 font-normal">Keine kompatible GPU erkannt</span>
            </div>
          )}
        </div>

        {/* Docker Overview Summary */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <span className="text-xs font-medium text-neutral-400 block mb-2">Docker Engine</span>
          <div className="grid grid-cols-3 gap-2 mt-2 text-center font-mono">
            <div className="bg-[#121212] p-2 rounded border border-[#242424]">
              <span className="text-[11px] text-neutral-500 block font-normal font-sans">Laufend</span>
              <span className="text-sm font-semibold tabular-nums text-emerald-400">{counts.running}</span>
            </div>
            <div className="bg-[#121212] p-2 rounded border border-[#242424]">
              <span className="text-[11px] text-neutral-500 block font-normal font-sans">Gestoppt</span>
              <span className="text-sm font-semibold tabular-nums text-neutral-400">{counts.stopped}</span>
            </div>
            <div className="bg-[#121212] p-2 rounded border border-[#242424]">
              <span className="text-[11px] text-neutral-500 block font-normal font-sans">Problem</span>
              <span className="text-sm font-semibold tabular-nums text-rose-400">{counts.unhealthy}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Live Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* CPU & RAM Live Chart */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold text-white tracking-tight">CPU & RAM Verlauf (Echtzeit)</h3>
              <p className="text-xs text-neutral-500 font-normal">Live-Abtastung im Sekundentakt</p>
            </div>
            <div className="flex items-center gap-4 text-xs font-medium font-mono">
              <span className="flex items-center gap-1.5 text-blue-400">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-400" /> CPU %
              </span>
              <span className="flex items-center gap-1.5 text-purple-400">
                <span className="w-1.5 h-1.5 rounded-full bg-purple-400" /> RAM %
              </span>
            </div>
          </div>
          <div className="h-52 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="ramGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#a855f7" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#a855f7" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" stroke="#525252" fontSize={10} tickLine={false} />
                <YAxis domain={[0, 100]} stroke="#525252" fontSize={10} tickLine={false} unit="%" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#141414', borderColor: '#282828', borderRadius: '0.375rem', fontSize: '11px', color: '#f5f5f5' }}
                />
                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fillOpacity={1} fill="url(#cpuGrad)" strokeWidth={1.5} isAnimationActive={false} />
                <Area type="monotone" dataKey="ram" stroke="#a855f7" fillOpacity={1} fill="url(#ramGrad)" strokeWidth={1.5} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Network Live Chart */}
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold text-white tracking-tight">Netzwerk-Durchsatz (Echtzeit)</h3>
              <p className="text-xs text-neutral-500 font-normal">RX / TX Bandbreite</p>
            </div>
            <div className="flex items-center gap-4 text-xs font-medium font-mono">
              <span className="flex items-center gap-1.5 text-emerald-400">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" /> Download
              </span>
              <span className="flex items-center gap-1.5 text-amber-400">
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400" /> Upload
              </span>
            </div>
          </div>
          <div className="h-52 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="rxGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="txGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" stroke="#525252" fontSize={10} tickLine={false} />
                <YAxis stroke="#525252" fontSize={10} tickLine={false} tickFormatter={(v) => formatBytes(v)} />
                <Tooltip
                  formatter={(val: any) => formatBps(val)}
                  contentStyle={{ backgroundColor: '#141414', borderColor: '#282828', borderRadius: '0.375rem', fontSize: '11px', color: '#f5f5f5' }}
                />
                <Area type="monotone" dataKey="rx" stroke="#10b981" fillOpacity={1} fill="url(#rxGrad)" strokeWidth={1.5} isAnimationActive={false} />
                <Area type="monotone" dataKey="tx" stroke="#f59e0b" fillOpacity={1} fill="url(#txGrad)" strokeWidth={1.5} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Active Containers (Direct Canvas Table, Tailscale style) */}
      <div className="pt-2">
        <div className="flex items-center justify-between mb-3">
          <div>
            <h3 className="text-base font-semibold text-white tracking-tight">Aktive Container ({counts.running})</h3>
            <p className="text-xs text-neutral-400 font-normal">Live-Statistiken aller aktuell ausgeführten Instanzen</p>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold">
                <th className="pb-3 px-3 font-semibold">Container</th>
                <th className="pb-3 px-3 font-semibold">Status</th>
                <th className="pb-3 px-3 font-semibold">CPU</th>
                <th className="pb-3 px-3 font-semibold">RAM</th>
                <th className="pb-3 px-3 font-semibold">Netzwerk RX/TX</th>
                <th className="pb-3 px-3 font-semibold">Uptime</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1f1f1f]">
              {containers.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-neutral-500 font-normal">
                    Keine Container auf diesem Server gefunden.
                  </td>
                </tr>
              ) : (
                containers.slice(0, 10).map((c) => (
                  <tr
                    key={c.id}
                    onClick={() => onSelectContainer(c.id)}
                    className="hover:bg-white/[0.02] cursor-pointer transition-colors"
                  >
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-white text-[13px] hover:underline">{c.name}</span>
                        <span className="text-[10px] text-neutral-400 font-mono px-1.5 py-0.5 rounded bg-[#1c1c1c] border border-[#2a2a2a]">{c.shortId}</span>
                      </div>
                    </td>
                    <td className="py-3 px-3">
                      <StatusBadge status={c.state} />
                    </td>
                    <td className="py-3 px-3 font-mono text-neutral-300 tabular-nums">
                      {c.stats?.cpuPercent.value !== null && c.stats?.cpuPercent.value !== undefined
                        ? formatPercent(c.stats.cpuPercent.value)
                        : '—'}
                    </td>
                    <td className="py-3 px-3 font-mono text-neutral-300 tabular-nums">
                      {c.stats?.memUsage.value !== null && c.stats?.memUsage.value !== undefined
                        ? `${formatBytes(c.stats.memUsage.value)} (${formatPercent(c.stats.memPercent.value)})`
                        : '—'}
                    </td>
                    <td className="py-3 px-3 font-mono text-neutral-400 tabular-nums">
                      ↓ {formatBps(c.stats?.netRxBps.value)} · ↑ {formatBps(c.stats?.netTxBps.value)}
                    </td>
                    <td className="py-3 px-3 font-mono text-neutral-400">
                      {c.startedAt ? formatUptime(Math.floor((Date.now() - new Date(c.startedAt).getTime()) / 1000)) : '—'}
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

