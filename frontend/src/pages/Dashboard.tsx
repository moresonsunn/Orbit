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
  ExternalLink,
  ChevronDown,
  ChevronUp,
  SlidersHorizontal,
  X,
  Check,
  Eye,
  EyeOff,
} from 'lucide-react';
import { MetricCard } from '../components/common/MetricCard';
import { StatusBadge } from '../components/common/StatusBadge';
import { Container, HostMetrics, GPUTelemetry, SystemInfoResponse } from '../types';
import { formatBytes, formatBps, formatPercent, formatUptime } from '../lib/utils';
import { wsClient } from '../lib/ws';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import {
  loadDashboardPrefs,
  saveDashboardPrefs,
  PRESET_CONFIGS,
  DashboardPreferences,
  DashboardPreset,
} from '../lib/dashboardConfig';

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
  const [showCores, setShowCores] = useState(false);
  const [prefs, setPrefs] = useState<DashboardPreferences>(loadDashboardPrefs);
  const [showConfigModal, setShowConfigModal] = useState(false);

  const updatePref = <K extends keyof DashboardPreferences>(key: K, value: DashboardPreferences[K]) => {
    setPrefs((prev) => {
      const next = { ...prev, [key]: value };
      saveDashboardPrefs(next);
      return next;
    });
  };

  const setPreset = (preset: DashboardPreset) => {
    const next = { ...PRESET_CONFIGS[preset] };
    setPrefs(next);
    saveDashboardPrefs(next);
  };

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

  const thermalSensors = metrics?.thermal?.sensors || [];
  const cpuSensors = thermalSensors.filter((s) => s.sensorType === 'cpu');
  const nvmeSensors = thermalSensors.filter((s) => s.sensorType === 'nvme');
  const mbSensors = thermalSensors.filter((s) => s.sensorType === 'motherboard');
  const hasSensors = thermalSensors.length > 0;

  return (
    <div className={`animate-in fade-in duration-200 font-sans ${prefs.compactMode ? 'space-y-4' : 'space-y-6'}`}>
      {/* Page Header (Tailscale style) */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Dashboard</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Zentrale Host-Telemetrie und Docker-Ressourcen im Sekundentakt.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {/* Quick Preset Selector */}
          <div className="hidden md:flex items-center bg-[#181818] border border-[#282828] p-0.5 rounded-lg text-xs">
            {(['minimal', 'balanced', 'pro'] as const).map((p) => (
              <button
                key={p}
                onClick={() => setPreset(p)}
                className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                  prefs.preset === p
                    ? 'bg-white/10 text-white shadow-xs'
                    : 'text-neutral-400 hover:text-white'
                }`}
              >
                {p === 'minimal' ? 'Kompakt' : p === 'balanced' ? 'Standard' : 'Pro Sysadmin'}
              </button>
            ))}
          </div>

          <button
            onClick={() => setShowConfigModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-[#181818] border border-[#282828] hover:bg-[#222222] text-neutral-300 rounded-md text-xs font-medium transition-colors shadow-xs"
            title="Widgets und Dashboard-Elemente anpassen"
          >
            <SlidersHorizontal className="w-3.5 h-3.5" />
            <span>Anpassen</span>
          </button>
        </div>
      </div>

      {/* Upper Information Grid */}
      {prefs.showSummary && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* CPU Card */}
          <MetricCard
            title="CPU Gesamt"
            headerRight={
              metrics?.cpu.perCore && metrics.cpu.perCore.length > 0 ? (
                <button
                  onClick={() => setShowCores(!showCores)}
                  className={`text-[10px] px-2 py-0.5 rounded transition-colors flex items-center gap-1 ${
                    showCores
                      ? 'bg-white/10 text-white border border-neutral-600'
                      : 'bg-[#202020] text-neutral-400 border border-[#2c2c2c] hover:text-white hover:bg-[#262626]'
                  }`}
                  title="Kerne & Threads detailliert anzeigen"
                >
                  <span>{showCores ? 'Verbergen' : `${metrics.cpu.threads} Kerne`}</span>
                  {showCores ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                </button>
              ) : null
            }
            gauge={metrics?.cpu.totalPercent}
            formattedValue={formatPercent(metrics?.cpu.totalPercent.value)}
            progressPercent={metrics?.cpu.totalPercent.value ?? undefined}
            subtitle={
              <div className="space-y-0.5 truncate">
                <span className="font-medium text-neutral-200 block truncate" title={metrics?.cpu.modelName || 'Host Prozessor'}>
                  {metrics?.cpu.modelName || 'Host Prozessor'}
                </span>
                <span className="text-neutral-500 text-[11px] block font-mono truncate">
                  {metrics?.cpu.cores ?? '—'} Kerne · {metrics?.cpu.threads ?? '—'} Threads
                  {metrics?.cpu.freqMHz.value
                    ? ` · ${(metrics.cpu.freqMHz.value >= 1000
                        ? (metrics.cpu.freqMHz.value / 1000).toFixed(2) + ' GHz'
                        : metrics.cpu.freqMHz.value + ' MHz')}`
                    : ''}
                </span>
              </div>
            }
          />

          {/* RAM Card */}
          <MetricCard
            title="Arbeitsspeicher"
            gauge={metrics?.memory.usedPercent}
            formattedValue={formatPercent(metrics?.memory.usedPercent.value)}
            progressPercent={metrics?.memory.usedPercent.value ?? undefined}
            subtitle={
              <div className="space-y-0.5 truncate">
                <span className="font-medium text-neutral-200 block">
                  {formatBytes(metrics?.memory.usedBytes)} von {formatBytes(metrics?.memory.totalBytes)}
                </span>
                <span className="text-neutral-500 text-[11px] block font-mono truncate">
                  Frei: {formatBytes(metrics?.memory.availableBytes || metrics?.memory.freeBytes)}
                  {metrics?.memory.cachedBytes ? ` · Cache: ${formatBytes(metrics.memory.cachedBytes)}` : ''}
                </span>
              </div>
            }
          />

          {/* Network RX/TX Card */}
          <MetricCard
            title="Netzwerk-Durchsatz"
            formattedValue={formatBps((metrics?.network.rxBps.value ?? 0) + (metrics?.network.txBps.value ?? 0))}
            subtitle={
              <div className="space-y-0.5 truncate">
                <span className="font-medium text-neutral-200 block">
                  ↓ {formatBps(metrics?.network.rxBps.value)} · ↑ {formatBps(metrics?.network.txBps.value)}
                </span>
                <span className="text-neutral-500 text-[11px] block font-mono truncate">
                  Gesamt: {formatBytes(metrics?.network.totalRxBytes)} RX · {formatBytes(metrics?.network.totalTxBytes)} TX
                </span>
              </div>
            }
          />

          {/* Storage Card */}
          <MetricCard
            title="Festplattenspeicher"
            gauge={metrics?.storage.usedPercent}
            formattedValue={formatPercent(metrics?.storage.usedPercent.value)}
            progressPercent={metrics?.storage.usedPercent.value ?? undefined}
            subtitle={
              <div className="space-y-0.5 truncate">
                <span className="font-medium text-neutral-200 block">
                  {formatBytes(metrics?.storage.usedBytes)} von {formatBytes(metrics?.storage.totalBytes)}
                </span>
                <span className="text-neutral-500 text-[11px] block font-mono truncate">
                  I/O: ↓ {formatBps(metrics?.storage.readBps.value)} · ↑ {formatBps(metrics?.storage.writeBps.value)}
                </span>
              </div>
            }
          />
        </div>
      )}

      {/* Expandable Per-Core Grid (Beszel/htop style) */}
      {showCores && metrics?.cpu.perCore && metrics.cpu.perCore.length > 0 && (
        <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs animate-in slide-in-from-top-2 duration-200">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h4 className="text-xs font-semibold text-white tracking-tight">CPU Kern-Auslastung</h4>
              <p className="text-[11px] text-neutral-500">Live-Abtastung aller {metrics.cpu.perCore.length} logischen Prozessorkerne</p>
            </div>
            <span className="text-[11px] font-mono text-neutral-400">
              Durchschnitt: {formatPercent(metrics.cpu.totalPercent.value)}
            </span>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-8 lg:grid-cols-16 gap-2">
            {metrics.cpu.perCore.map((core, idx) => {
              const val = core.value ?? 0;
              return (
                <div key={idx} className="bg-[#121212] border border-[#262626] rounded p-2 text-center">
                  <div className="flex items-center justify-between text-[10px] font-mono mb-1 text-neutral-400">
                    <span>C{idx}</span>
                    <span className={val > 80 ? 'text-rose-400 font-bold' : 'text-neutral-300'}>{val.toFixed(0)}%</span>
                  </div>
                  <div className="w-full bg-[#202020] h-1.5 rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all duration-200 rounded-full ${
                        val > 80 ? 'bg-rose-500' : val > 60 ? 'bg-amber-500' : 'bg-emerald-500'
                      }`}
                      style={{ width: `${Math.min(100, Math.max(0, val))}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Secondary Metrics (Quiet, cohesive bar) */}
      {(prefs.showDockerOverview || prefs.showLoadAvg || (prefs.showThermals && (hasSensors || prefs.preset === 'pro')) || (prefs.showGPU && (primaryGPU || prefs.preset === 'pro'))) && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Docker Status */}
          {prefs.showDockerOverview && (
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
          )}

          {/* Load Average */}
          {prefs.showLoadAvg && (
            <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
              <div className="flex items-center justify-between text-neutral-400 mb-2">
                <span className="text-xs font-medium">Load Average</span>
                <span className="text-[11px] text-neutral-500 font-mono">1m · 5m · 15m</span>
              </div>
              <div className="flex items-center justify-between text-white mt-3">
                <div>
                  <span className="text-[10px] text-neutral-500 block font-normal font-sans">1 min</span>
                  <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg1.value?.toFixed(2) ?? '—'}</span>
                </div>
                <div>
                  <span className="text-[10px] text-neutral-500 block font-normal font-sans">5 min</span>
                  <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg5.value?.toFixed(2) ?? '—'}</span>
                </div>
                <div>
                  <span className="text-[10px] text-neutral-500 block font-normal font-sans">15 min</span>
                  <span className="text-base font-semibold tabular-nums text-white font-mono">{metrics?.cpu.loadAvg15.value?.toFixed(2) ?? '—'}</span>
                </div>
              </div>
            </div>
          )}

          {/* Systemtemperaturen (Multi-Sensor) */}
          {prefs.showThermals && (
            <div className="p-4 rounded-lg bg-[#181818] border border-[#282828] shadow-xs">
              <div className="flex items-center justify-between text-neutral-400 mb-2">
                <span className="text-xs font-medium">Systemtemperatur</span>
                <Thermometer className="w-3.5 h-3.5 text-neutral-500" />
              </div>
              {hasSensors ? (
                <div className="space-y-2 mt-1">
                  <div className="flex items-baseline justify-between">
                    <span className="text-xl font-semibold text-white font-mono">
                      {metrics?.thermal?.cpuTemp.value ? `${metrics.thermal.cpuTemp.value} °C` : '—'}
                    </span>
                    <span className="text-[11px] text-neutral-400 font-mono">CPU Package</span>
                  </div>
                  <div className="flex flex-wrap gap-1.5 pt-1">
                    {nvmeSensors.slice(0, 2).map((s, idx) => (
                      <span key={idx} className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#121212] border border-[#262626] text-neutral-300">
                        {s.name}: {s.tempC.value}°C
                      </span>
                    ))}
                    {mbSensors.slice(0, 1).map((s, idx) => (
                      <span key={idx} className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#121212] border border-[#262626] text-neutral-300">
                        Board: {s.tempC.value}°C
                      </span>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="py-2 text-xs text-neutral-500 font-mono">
                  <span>Keine Host-Sensoren erkannt</span>
                  <span className="block text-[10px] text-neutral-600 mt-0.5">Uptime: {formatUptime(metrics?.system.uptimeSeconds.value)}</span>
                </div>
              )}
            </div>
          )}

          {/* GPU Telemetry Preview (Smart Auto-Hide when no GPU) */}
          {prefs.showGPU && (
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
          )}
        </div>
      )}

      {/* Live Charts Section (Togglable) */}
      {prefs.showCharts && (
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
                <th className="pb-3 px-3 font-semibold">Web / Ports</th>
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
                  <td colSpan={7} className="py-8 text-center text-neutral-500 font-normal">
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
                    <td className="py-3 px-3" onClick={(e) => e.stopPropagation()}>
                      <div className="flex flex-wrap gap-1 items-center">
                        {c.ports.filter((p) => p.publicPort).slice(0, 2).map((p, idx) => (
                          <a
                            key={idx}
                            href={`http://${window.location.hostname}:${p.publicPort}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-mono bg-[#1c1c1c] border border-[#2a2a2a] text-blue-400 hover:border-blue-500/70 hover:bg-blue-950/20 transition-colors"
                            title={`Web-App in neuem Tab öffnen: :${p.publicPort}`}
                          >
                            <span>:{p.publicPort}</span>
                            <ExternalLink className="w-2.5 h-2.5 opacity-70" />
                          </a>
                        ))}
                        {c.ports.filter((p) => p.publicPort).length === 0 && (
                          <span className="text-neutral-600 font-mono text-[11px]">—</span>
                        )}
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

