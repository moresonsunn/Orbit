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
      <div className="flex items-center justify-center h-64 text-neutral-400 font-mono text-xs">
        Lade GPU-Telemetrie...
      </div>
    );
  }

  if (!telemetry.available || !telemetry.devices || telemetry.devices.length === 0) {
    return (
      <div className="p-8 rounded-lg bg-[#181818] border border-[#282828] text-center space-y-3 max-w-xl mx-auto my-12 animate-in fade-in font-sans">
        <div className="inline-flex items-center justify-center w-10 h-10 rounded-md bg-[#222222] border border-[#2e2e2e] text-neutral-400">
          <Cpu className="w-5 h-5" />
        </div>
        <h3 className="text-base font-semibold text-white tracking-tight">Keine kompatible GPU erkannt</h3>
        <p className="text-xs text-neutral-400 leading-relaxed font-sans">
          {telemetry.message ||
            'Weder NVIDIA NVML noch AMD/Intel DRM Schnittstellen sind auf diesem Host zugänglich. Das Dashboard und alle Container-Funktionen laufen normal weiter.'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">GPU Monitoring</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Live-Telemetrie via NVML / Direct Rendering Manager ({telemetry.devices.length} Karte(n) aktiv).
          </p>
        </div>
      </div>

      {telemetry.devices.map((dev) => (
        <div key={dev.index} className="space-y-4 p-5 rounded-lg bg-[#181818] border border-[#282828]">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-4 border-b border-[#242424]">
            <div>
              <h4 className="text-base font-semibold text-white">{dev.model}</h4>
              <p className="text-xs text-neutral-400 font-mono mt-0.5">
                Hersteller: {dev.vendor.toUpperCase()} • Treiber: {dev.driverVersion}
              </p>
            </div>
            <span className="px-2 py-0.5 rounded text-xs font-mono font-medium bg-[#222222] text-neutral-300 border border-[#2e2e2e] self-start sm:self-auto">
              GPU #{dev.index}
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <MetricCard
              title="GPU Auslastung"
              gauge={dev.utilizationPercent}
              formattedValue={formatPercent(dev.utilizationPercent.value)}
              progressPercent={dev.utilizationPercent.value ?? undefined}
            />
            <MetricCard
              title="VRAM Speicher"
              gauge={dev.memoryPercent}
              formattedValue={formatPercent(dev.memoryPercent.value)}
              progressPercent={dev.memoryPercent.value ?? undefined}
              subtitle={`${formatBytes(dev.memoryUsedBytes.value)} / ${formatBytes(dev.memoryTotalBytes.value)}`}
            />
            <MetricCard
              title="Temperatur"
              gauge={dev.temperatureCelsius}
              formattedValue={dev.temperatureCelsius.value ? `${dev.temperatureCelsius.value} °C` : '—'}
            />
            <MetricCard
              title="Leistungsaufnahme"
              gauge={dev.powerDrawWatts}
              formattedValue={dev.powerDrawWatts.value ? `${dev.powerDrawWatts.value} W` : '—'}
              subtitle={dev.powerLimitWatts.value ? `Limit: ${dev.powerLimitWatts.value} W` : undefined}
            />
          </div>

          {/* Running GPU Processes */}
          {dev.processes && dev.processes.length > 0 && (
            <div className="mt-4 pt-4 border-t border-[#242424]">
              <h5 className="text-xs font-semibold text-neutral-300 uppercase tracking-wider mb-3">
                Container GPU-Prozesszuordnung
              </h5>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs font-mono">
                  <thead>
                    <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider font-sans">
                      <th className="pb-2 font-medium">PID</th>
                      <th className="pb-2 font-medium">Prozess</th>
                      <th className="pb-2 font-medium">Zugehöriger Container</th>
                      <th className="pb-2 font-medium text-right">VRAM Belegung</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#202020]">
                    {dev.processes.map((proc, i) => (
                      <tr key={i} className="hover:bg-white/[0.02]">
                        <td className="py-2.5 text-neutral-300">{proc.pid}</td>
                        <td className="py-2.5 text-neutral-200">{proc.processName || '—'}</td>
                        <td className="py-2.5 text-neutral-300 font-sans">
                          {proc.containerName ? (
                            <span className="text-emerald-400 font-medium">{proc.containerName}</span>
                          ) : (
                            <span className="text-neutral-500">Host-Prozess</span>
                          )}
                        </td>
                        <td className="py-2.5 text-right text-neutral-200">
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

