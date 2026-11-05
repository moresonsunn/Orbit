import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, ShieldAlert, Bell, Clock, Database, Check } from 'lucide-react';
import { Settings, AuthStatus } from '../types';
import { api } from '../lib/api';

interface SettingsPageProps {
  authStatus?: AuthStatus;
}

export const SettingsPage: React.FC<SettingsPageProps> = ({ authStatus }) => {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isAdmin = authStatus?.user?.role === 'admin';

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const data = await api.getSettings();
      setSettings(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!settings || !isAdmin) return;
    setSaving(true);
    setError(null);

    try {
      await api.updateSettings(settings);
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2500);
    } catch (err: any) {
      setError(err.message || 'Speichern fehlgeschlagen');
    } finally {
      setSaving(false);
    }
  };

  if (loading || !settings) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 font-mono text-xs">
        Lade Einstellungen...
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-4xl animate-in fade-in duration-200">
      <div className="flex items-center justify-between p-4 rounded-lg bg-[#151b26] border border-[#232d3f]">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-[#0e121b] border border-[#232d3f] text-blue-400">
            <SettingsIcon className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100 tracking-tight">Plattform-Einstellungen</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Abtastraten, Warnschwellen, Datenaufbewahrung und Feature-Flags
            </p>
          </div>
        </div>

        {isAdmin && (
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-md text-xs font-medium transition-all shadow-xs"
          >
            {savedSuccess ? <Check className="w-3.5 h-3.5" /> : <Save className="w-3.5 h-3.5" />}
            <span>{savedSuccess ? 'Gespeichert!' : saving ? 'Wird gespeichert...' : 'Änderungen speichern'}</span>
          </button>
        )}
      </div>

      {error && (
        <div className="p-3 bg-rose-950/40 border border-rose-800/60 rounded-md text-xs text-rose-300">
          {error}
        </div>
      )}

      <form onSubmit={handleSave} className="space-y-6">
        {/* Feature Flags Overview */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] space-y-3">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200 uppercase tracking-wider font-mono">
            <ShieldAlert className="w-4 h-4 text-blue-400" />
            <span>Aktive Feature-Flags (Host-Umgebung)</span>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs font-mono">
            <div className="p-3 bg-[#0e121b] rounded-md border border-[#232d3f]">
              <span className="text-slate-400 block text-[10px]">Aktionen (Actions)</span>
              <span
                className={`font-semibold ${
                  authStatus?.features.enableActions ? 'text-emerald-400' : 'text-slate-500'
                }`}
              >
                {authStatus?.features.enableActions ? 'Aktiviert' : 'Gesperrt (Read-only)'}
              </span>
            </div>
            <div className="p-3 bg-[#0e121b] rounded-md border border-[#232d3f]">
              <span className="text-slate-400 block text-[10px]">Destruktiv (Delete/Prune)</span>
              <span
                className={`font-semibold ${
                  authStatus?.features.enableDestructive ? 'text-emerald-400' : 'text-slate-500'
                }`}
              >
                {authStatus?.features.enableDestructive ? 'Aktiviert' : 'Gesperrt'}
              </span>
            </div>
            <div className="p-3 bg-[#0e121b] rounded-md border border-[#232d3f]">
              <span className="text-slate-400 block text-[10px]">eBPF Modus</span>
              <span className="text-blue-400 font-semibold uppercase">{authStatus?.features.ebpfMode || 'auto'}</span>
            </div>
            <div className="p-3 bg-[#0e121b] rounded-md border border-[#232d3f]">
              <span className="text-slate-400 block text-[10px]">GPU Modus</span>
              <span className="text-indigo-400 font-semibold uppercase">{authStatus?.features.gpuMode || 'auto'}</span>
            </div>
          </div>
        </div>

        {/* Intervals */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200 uppercase tracking-wider font-mono">
            <Clock className="w-4 h-4 text-blue-400" />
            <span>Echtzeit-Abtastintervalle (Millisekunden)</span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="block text-slate-300 mb-1">Host-Metriken Intervall (ms)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.intervals.hostMs}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    intervals: { ...settings.intervals, hostMs: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Container-Metriken Intervall (ms)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.intervals.containerMs}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    intervals: { ...settings.intervals, containerMs: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">GPU Telemetrie Intervall (ms)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.intervals.gpuMs}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    intervals: { ...settings.intervals, gpuMs: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Netflow Topologie Intervall (ms)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.intervals.netflowMs}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    intervals: { ...settings.intervals, netflowMs: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
          </div>
        </div>

        {/* Alert Thresholds */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200 uppercase tracking-wider font-mono">
            <Bell className="w-4 h-4 text-blue-400" />
            <span>Schwellenwerte für Warnungen (Alerts)</span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            <div>
              <label className="block text-slate-300 mb-1">Host CPU Warnschwelle (%)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.alerts.hostCpu.value}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    alerts: {
                      ...settings.alerts,
                      hostCpu: { ...settings.alerts.hostCpu, value: parseFloat(e.target.value) },
                    },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Host RAM Warnschwelle (%)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.alerts.hostMemory.value}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    alerts: {
                      ...settings.alerts,
                      hostMemory: { ...settings.alerts.hostMemory, value: parseFloat(e.target.value) },
                    },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Festplattenplatz Warnschwelle (%)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.alerts.diskUsage.value}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    alerts: {
                      ...settings.alerts,
                      diskUsage: { ...settings.alerts.diskUsage, value: parseFloat(e.target.value) },
                    },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">GPU Temperatur Warnschwelle (°C)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.alerts.gpuTemp.value}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    alerts: {
                      ...settings.alerts,
                      gpuTemp: { ...settings.alerts.gpuTemp, value: parseFloat(e.target.value) },
                    },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 font-mono focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
          </div>
        </div>

        {/* Retention */}
        <div className="p-4 rounded-lg bg-[#151b26] border border-[#232d3f] space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200 uppercase tracking-wider font-mono">
            <Database className="w-4 h-4 text-blue-400" />
            <span>SQLite Historien-Aufbewahrung (Retention)</span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
            <div>
              <label className="block text-slate-300 mb-1">Rohdaten (Stunden)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.retention.rawHours}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    retention: { ...settings.retention, rawHours: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">5-Minuten Rollup (Tage)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.retention.days5m}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    retention: { ...settings.retention, days5m: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">1-Stunden Rollup (Tage)</label>
              <input
                type="number"
                disabled={!isAdmin}
                value={settings.retention.days1h}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    retention: { ...settings.retention, days1h: parseInt(e.target.value, 10) },
                  })
                }
                className="w-full px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-slate-200 focus:outline-none focus:border-blue-500 disabled:opacity-50"
              />
            </div>
          </div>
        </div>
      </form>
    </div>
  );
};

