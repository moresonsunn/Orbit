import React, { useState, useEffect } from 'react';
import { Cpu, Zap, Thermometer, Activity, Layers, AlertCircle } from 'lucide-react';
import { GPUTelemetry } from '../types';
import { api } from '../lib/api';
import { wsClient } from '../lib/ws';
import { formatBytes, formatPercent } from '../lib/utils';
import { MetricCard } from '../components/common/MetricCard';

export const GPUPage: React.FC = () => {
  const [telemetry, setTelemetry] = useState<GPUTelemetry | null>(null);
  const [loading, setLoading] = useState(true);

  const loadGPU = async () => {
    try {
      const data = await api.getGPU();
      setTelemetry(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadGPU();

    const unsub = wsClient.subscribe('gpu', (data: GPUTelemetry) => {
      setTelemetry(data);
    });

    return () => unsub();
  }, []);

  if (loading || !telemetry) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 font-mono text-xs">
        Lade GPU-Telemetrie...
      </div>
    );
  }

  if (!telemetry.available || !telemetry.devices || telemetry.devices.length === 0) {
    return (
      <div className="p-8 rounded-lg bg-[#151b26] border border-[#232d3f] text-center space-y-3 max-w-xl mx-auto my-12 animate-in fade-in">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-[#0e121b] border border-[#232d3f] text-slate-400">
          <Cpu className="w-6 h-6" />
        </div>
        <h3 className="text-base font-semibold text-slate-100 tracking-tight">Keine kompatible GPU erkannt</h3>
        <p className="text-xs text-slate-400 leading-relaxed font-mono">
          {telemetry.message ||
            'Weder NVIDIA NVML noch AMD/Intel DRM Schnittstellen sind auf diesem Host zugänglich. Das Dashboard und alle Container-Funktionen laufen normal weiter.'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex items-center justify-between p-4 rounded-lg bg-[#151b26] border border-[#232d3f]">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-[#0e121b] border border-[#232d3f] text-blue-400">
            <Cpu className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100 tracking-tight">
              GPU Monitoring ({telemetry.devices.length} Karte(n) aktiv)
            </h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Live-Telemetrie via NVML / Direct Rendering Manager
            </p>
          </div>
        </div>
      </div>

      {telemetry.devices.map((dev) => (
        <div key={dev.index} className="space-y-4 p-5 rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-4 border-b border-[#232d3f]">
            <div>
              <h4 className="text-base font-semibold text-slate-100">{dev.model}</h4>
              <p className="text-xs text-slate-400 font-mono mt-0.5">
                Hersteller: {dev.vendor.toUpperCase()} • Treiber: {dev.driverVersion}
              </p>
            </div>
            <span className="px-2.5 py-0.5 rounded text-xs font-mono font-medium bg-[#0e121b] text-blue-400 border border-[#232d3f] self-start sm:self-auto">
              GPU #{dev.index}
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <MetricCard
              title="GPU Auslastung"
              gauge={dev.utilizationPercent}
              formattedValue={formatPercent(dev.utilizationPercent.value)}
              progressPercent={dev.utilizationPercent.value ?? undefined}
              icon={<Zap className="w-4 h-4 text-blue-400" />}
            />
            <MetricCard
              title="VRAM Speicher"
              gauge={dev.memoryPercent}
              formattedValue={formatPercent(dev.memoryPercent.value)}
              progressPercent={dev.memoryPercent.value ?? undefined}
              subtitle={`${formatBytes(dev.memoryUsedBytes.value)} / ${formatBytes(dev.memoryTotalBytes.value)}`}
              icon={<Layers className="w-4 h-4 text-indigo-400" />}
            />
            <MetricCard
              title="Temperatur"
              gauge={dev.temperatureCelsius}
              formattedValue={dev.temperatureCelsius.value ? `${dev.temperatureCelsius.value} °C` : '—'}
              icon={<Thermometer className="w-4 h-4 text-amber-400" />}
            />
            <MetricCard
              title="Leistungsaufnahme"
              gauge={dev.powerDrawWatts}
              formattedValue={dev.powerDrawWatts.value ? `${dev.powerDrawWatts.value} W` : '—'}
              subtitle={dev.powerLimitWatts.value ? `Limit: ${dev.powerLimitWatts.value} W` : undefined}
              icon={<Activity className="w-4 h-4 text-emerald-400" />}
            />
          </div>

          {/* Running GPU Processes */}
          {dev.processes && dev.processes.length > 0 && (
            <div className="mt-4 pt-4 border-t border-[#232d3f]">
              <h5 className="text-xs font-semibold text-slate-300 uppercase tracking-wider font-mono mb-3">
                Container GPU-Prozesszuordnung
              </h5>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs font-mono">
                  <thead>
                    <tr className="border-b border-[#232d3f] text-slate-400 uppercase tracking-wider">
                      <th className="pb-2 font-medium">PID</th>
                      <th className="pb-2 font-medium">Prozess</th>
                      <th className="pb-2 font-medium">Zugehöriger Container</th>
                      <th className="pb-2 font-medium text-right">VRAM Belegung</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#232d3f]/40">
                    {dev.processes.map((proc, i) => (
                      <tr key={i} className="hover:bg-[#1a2230]">
                        <td className="py-2.5 text-blue-400">{proc.pid}</td>
                        <td className="py-2.5 text-slate-200">{proc.processName || '—'}</td>
                        <td className="py-2.5 text-slate-300 font-medium">
                          {proc.containerName ? (
                            <span className="text-emerald-400 font-semibold">{proc.containerName}</span>
                          ) : (
                            <span className="text-slate-500">Host-Prozess</span>
                          )}
                        </td>
                        <td className="py-2.5 text-right text-slate-200">
                          {formatBytes(proc.usedMemoryBytes.value)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

