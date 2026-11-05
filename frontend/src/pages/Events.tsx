import React, { useState, useEffect } from 'react';
import { Activity, AlertTriangle, AlertCircle, Info, RefreshCw } from 'lucide-react';
import { DockerEvent } from '../types';
import { api } from '../lib/api';
import { wsClient } from '../lib/ws';
import { formatDate } from '../lib/utils';

export const EventsPage: React.FC = () => {
  const [events, setEvents] = useState<DockerEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterSeverity, setFilterSeverity] = useState<'all' | 'warning' | 'critical'>('all');

  const loadEvents = async () => {
    try {
      const data = await api.getEvents(100);
      setEvents(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadEvents();

    const unsub = wsClient.subscribe('events', (newEvent: DockerEvent) => {
      setEvents((prev) => [newEvent, ...prev.slice(0, 99)]);
    });

    return () => unsub();
  }, []);

  const filtered = events.filter((ev) => {
    if (filterSeverity === 'warning') return ev.severity === 'warning' || ev.severity === 'critical';
    if (filterSeverity === 'critical') return ev.severity === 'critical';
    return true;
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-2xl bg-slate-900/60 border border-slate-800/80">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-cyan-950/60 border border-cyan-800/60 text-cyan-400">
            <Activity className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-white tracking-tight">Docker Activity Feed</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Kontinuierlich erfasste Container- und System-Ereignisse
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {(['all', 'warning', 'critical'] as const).map((sev) => (
            <button
              key={sev}
              onClick={() => setFilterSeverity(sev)}
              className={`px-3 py-1 rounded-lg text-xs font-medium capitalize transition-all ${
                filterSeverity === sev
                  ? 'bg-cyan-950 text-cyan-300 border border-cyan-800'
                  : 'text-slate-400 hover:text-white bg-slate-900'
              }`}
            >
              {sev === 'all' ? 'Alle Events' : sev === 'warning' ? 'Warnungen' : 'Kritisch'}
            </button>
          ))}
          <button
            onClick={loadEvents}
            className="p-1.5 rounded-lg bg-slate-800 text-slate-300 hover:text-white"
            title="Aktualisieren"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="overflow-hidden rounded-2xl bg-slate-900/60 border border-slate-800/80 shadow-sm">
        <div className="divide-y divide-slate-800/60 font-mono text-xs">
          {filtered.length === 0 ? (
            <div className="p-8 text-center text-slate-400">Keine Ereignisse vorhanden.</div>
          ) : (
            filtered.map((ev) => (
              <div key={ev.id || `${ev.actorId}-${ev.timestamp}`} className="p-4 hover:bg-slate-800/30 transition-colors flex items-start gap-3">
                <div className="mt-0.5 shrink-0">
                  {ev.severity === 'critical' ? (
                    <AlertCircle className="w-4 h-4 text-rose-400" />
                  ) : ev.severity === 'warning' ? (
                    <AlertTriangle className="w-4 h-4 text-amber-400" />
                  ) : (
                    <Info className="w-4 h-4 text-cyan-400" />
                  )}
                </div>

                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-bold text-white font-sans text-sm">{ev.actorName}</span>
                    <span className="text-[11px] text-slate-400 shrink-0">{formatDate(ev.timestamp)}</span>
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="px-2 py-0.5 rounded text-[10px] uppercase font-bold bg-slate-800 text-slate-300 border border-slate-700">
                      {ev.type}:{ev.action}
                    </span>
                    <span className="text-slate-400 truncate max-w-sm">ID: {ev.actorId.slice(0, 12)}</span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

