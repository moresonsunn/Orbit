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
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Upper Information Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* CPU Card */}
        <MetricCard
          title="CPU Gesamt"
          icon={<Cpu className="w-4 h-4 text-sky-400" />}
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
          icon={<Server className="w-4 h-4 text-indigo-400" />}
          gauge={metrics?.memory.usedPercent}
          formattedValue={formatPercent(metrics?.memory.usedPercent.value)}
          progressPercent={metrics?.memory.usedPercent.value ?? undefined}
          subtitle={`${formatBytes(metrics?.memory.usedBytes)} von ${formatBytes(metrics?.memory.totalBytes)}`}
        />

        {/* Network RX/TX Card */}
        <MetricCard
          title="Netzwerk-Durchsatz"
          icon={<Activity className="w-4 h-4 text-emerald-400" />}
          formattedValue={formatBps((metrics?.network.rxBps.value ?? 0) + (metrics?.network.txBps.value ?? 0))}
          subtitle={`↓ ${formatBps(metrics?.network.rxBps.value)} · ↑ ${formatBps(metrics?.network.txBps.value)}`}
        />

        {/* Storage Card */}
        <MetricCard
          title="Festplattenspeicher"
          icon={<HardDrive className="w-4 h-4 text-amber-400" />}
          gauge={metrics?.storage.usedPercent}
          formattedValue={formatPercent(metrics?.storage.usedPercent.value)}
          progressPercent={metrics?.storage.usedPercent.value ?? undefined}
          subtitle={`${formatBytes(metrics?.storage.usedBytes)} von ${formatBytes(metrics?.storage.totalBytes)}`}
        />
      </div>

      {/* Secondary Metrics & Docker Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Load Average */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Load Average</span>
            <div className="p-1 rounded bg-[#0e121b] border border-[#232d3f] text-slate-400">
              <Activity className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-center justify-between text-white mt-2">
            <div>
              <span className="text-[11px] text-slate-400 block font-normal">1 min</span>
              <span className="text-lg font-bold tabular-nums text-slate-100">{metrics?.cpu.loadAvg1.value?.toFixed(2) ?? '—'}</span>
            </div>
            <div>
              <span className="text-[11px] text-slate-400 block font-normal">5 min</span>
              <span className="text-lg font-bold tabular-nums text-slate-100">{metrics?.cpu.loadAvg5.value?.toFixed(2) ?? '—'}</span>
            </div>
            <div>
              <span className="text-[11px] text-slate-400 block font-normal">15 min</span>
              <span className="text-lg font-bold tabular-nums text-slate-100">{metrics?.cpu.loadAvg15.value?.toFixed(2) ?? '—'}</span>
            </div>
          </div>
        </div>

        {/* Temperature & Host Power */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Systemtemperatur</span>
            <div className="p-1 rounded bg-[#0e121b] border border-[#232d3f] text-slate-400">
              <Thermometer className="w-4 h-4" />
            </div>
          </div>
          <div className="flex items-baseline gap-2 mt-1">
            <span className="text-2xl font-bold tracking-tight text-white tabular-nums">
              {metrics?.cpu.tempC.value ? `${metrics.cpu.tempC.value} °C` : '—'}
            </span>
            <span className="text-xs text-slate-400 font-normal">CPU</span>
          </div>
          <p className="text-xs text-slate-400 mt-2 font-normal">
            Laufzeit: {formatUptime(metrics?.system.uptimeSeconds.value)}
          </p>
        </div>

        {/* GPU Telemetry Preview */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">GPU Telemetrie</span>
            <div className="p-1 rounded bg-[#0e121b] border border-[#232d3f] text-slate-400">
              <Zap className="w-4 h-4 text-blue-400" />
            </div>
          </div>
          {primaryGPU ? (
            <div>
              <div className="flex items-baseline gap-2 mt-1">
                <span className="text-xl font-bold tracking-tight text-white tabular-nums">
                  {formatPercent(primaryGPU.utilizationPercent.value)}
                </span>
                <span className="text-xs text-slate-400 font-normal">Auslastung</span>
              </div>
              <p className="text-xs text-slate-400 mt-2 font-normal truncate">
                {primaryGPU.model} {primaryGPU.temperatureCelsius.value ? `(${primaryGPU.temperatureCelsius.value}°C)` : ''}
              </p>
            </div>
          ) : (
            <div className="py-2">
              <span className="text-xs text-slate-400 font-normal">Keine kompatible GPU erkannt</span>
            </div>
          )}
        </div>

        {/* Docker Overview Summary */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Docker Engine</span>
            <div className="p-1 rounded bg-[#0e121b] border border-[#232d3f] text-slate-400">
              <Boxes className="w-4 h-4 text-blue-400" />
            </div>
          </div>
          <div className="grid grid-cols-3 gap-2 mt-2 text-center">
            <div className="bg-[#0e121b] p-2 rounded border border-[#232d3f]">
              <span className="text-[11px] text-slate-400 block font-normal">Laufend</span>
              <span className="text-base font-bold tabular-nums text-emerald-400">{counts.running}</span>
            </div>
            <div className="bg-[#0e121b] p-2 rounded border border-[#232d3f]">
              <span className="text-[11px] text-slate-400 block font-normal">Gestoppt</span>
              <span className="text-base font-bold tabular-nums text-slate-400">{counts.stopped}</span>
            </div>
            <div className="bg-[#0e121b] p-2 rounded border border-[#232d3f]">
              <span className="text-[11px] text-slate-400 block font-normal">Problem</span>
              <span className="text-base font-bold tabular-nums text-rose-400">{counts.unhealthy}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Live Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* CPU & RAM Live Chart */}
        <div className="p-5 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-bold text-white tracking-tight">CPU & RAM Verlauf (Echtzeit)</h3>
              <p className="text-xs text-slate-400 font-normal">Live-Abtastung im Sekundentakt</p>
            </div>
            <div className="flex items-center gap-4 text-xs font-medium">
              <span className="flex items-center gap-1.5 text-blue-400">
                <span className="w-2 h-2 rounded-full bg-blue-400" /> CPU %
              </span>
              <span className="flex items-center gap-1.5 text-indigo-400">
                <span className="w-2 h-2 rounded-full bg-indigo-400" /> RAM %
              </span>
            </div>
          </div>
          <div className="h-56 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="ramGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#818cf8" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#818cf8" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" stroke="#475569" fontSize={11} tickLine={false} />
                <YAxis domain={[0, 100]} stroke="#475569" fontSize={11} tickLine={false} unit="%" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#151b26', borderColor: '#232d3f', borderRadius: '0.5rem', fontSize: '12px' }}
                />
                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fillOpacity={1} fill="url(#cpuGrad)" strokeWidth={1.5} isAnimationActive={false} />
                <Area type="monotone" dataKey="ram" stroke="#818cf8" fillOpacity={1} fill="url(#ramGrad)" strokeWidth={1.5} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Network Live Chart */}
        <div className="p-5 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-bold text-white tracking-tight">Netzwerk-Durchsatz (Echtzeit)</h3>
              <p className="text-xs text-slate-400 font-normal">RX / TX Bandbreite</p>
            </div>
            <div className="flex items-center gap-4 text-xs font-medium">
              <span className="flex items-center gap-1.5 text-emerald-400">
                <span className="w-2 h-2 rounded-full bg-emerald-400" /> Download
              </span>
              <span className="flex items-center gap-1.5 text-amber-400">
                <span className="w-2 h-2 rounded-full bg-amber-400" /> Upload
              </span>
            </div>
          </div>
          <div className="h-56 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="rxGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="txGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.25} />
                    <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" stroke="#475569" fontSize={11} tickLine={false} />
                <YAxis stroke="#475569" fontSize={11} tickLine={false} tickFormatter={(v) => formatBytes(v)} />
                <Tooltip
                  formatter={(val: any) => formatBps(val)}
                  contentStyle={{ backgroundColor: '#151b26', borderColor: '#232d3f', borderRadius: '0.5rem', fontSize: '12px' }}
                />
                <Area type="monotone" dataKey="rx" stroke="#10b981" fillOpacity={1} fill="url(#rxGrad)" strokeWidth={1.5} isAnimationActive={false} />
                <Area type="monotone" dataKey="tx" stroke="#f59e0b" fillOpacity={1} fill="url(#txGrad)" strokeWidth={1.5} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Active Containers Quick Grid */}
      <div className="p-5 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-bold text-white tracking-tight">Laufende Container ({counts.running})</h3>
            <p className="text-xs text-slate-400 font-normal">Live-Statistiken aller aktiven Instanzen</p>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-[#232d3f] text-slate-400 uppercase tracking-wider text-[11px] font-semibold bg-[#0e121b]/60">
                <th className="py-2.5 px-3 font-semibold">Container</th>
                <th className="py-2.5 px-3 font-semibold">Status</th>
                <th className="py-2.5 px-3 font-semibold">CPU</th>
                <th className="py-2.5 px-3 font-semibold">RAM</th>
                <th className="py-2.5 px-3 font-semibold">Netzwerk RX/TX</th>
                <th className="py-2.5 px-3 font-semibold">Uptime</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#232d3f]/60">
              {containers.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-6 text-center text-slate-400 font-normal">
                    Keine Container auf diesem Server gefunden.
                  </td>
                </tr>
              ) : (
                containers.slice(0, 10).map((c) => (
                  <tr
                    key={c.id}
                    onClick={() => onSelectContainer(c.id)}
                    className="hover:bg-[#1a2230]/60 cursor-pointer transition-colors"
                  >
                    <td className="py-3 px-3 font-medium text-white flex items-center gap-2">
                      <span className="font-bold text-slate-100">{c.name}</span>
                      <span className="text-[10px] text-slate-400 px-1.5 py-0.5 rounded bg-[#0e121b] border border-[#232d3f]">{c.shortId}</span>
                    </td>
                    <td className="py-3 px-3">
                      <StatusBadge status={c.state} />
                    </td>
                    <td className="py-3 px-3 text-slate-300 tabular-nums">
                      {c.stats?.cpuPercent.value !== null && c.stats?.cpuPercent.value !== undefined
                        ? formatPercent(c.stats.cpuPercent.value)
                        : '—'}
                    </td>
                    <td className="py-3 px-3 text-slate-300 tabular-nums">
                      {c.stats?.memUsage.value !== null && c.stats?.memUsage.value !== undefined
                        ? `${formatBytes(c.stats.memUsage.value)} (${formatPercent(c.stats.memPercent.value)})`
                        : '—'}
                    </td>
                    <td className="py-3 px-3 text-slate-300 tabular-nums">
                      ↓ {formatBps(c.stats?.netRxBps.value)} · ↑ {formatBps(c.stats?.netTxBps.value)}
                    </td>
                    <td className="py-3 px-3 font-normal text-slate-400">
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

