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
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Events</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Kontinuierlich erfasste Docker- und System-Ereignisse.
          </p>
        </div>

        <div className="flex items-center gap-1.5 self-start sm:self-auto">
          {(['all', 'warning', 'critical'] as const).map((sev) => (
            <button
              key={sev}
              onClick={() => setFilterSeverity(sev)}
              className={`px-2.5 py-1 rounded-md text-xs font-medium capitalize transition-colors ${
                filterSeverity === sev
                  ? 'bg-white/10 text-white'
                  : 'text-neutral-400 hover:text-white hover:bg-white/[0.04]'
              }`}
            >
              {sev === 'all' ? 'Alle' : sev === 'warning' ? 'Warnungen' : 'Kritisch'}
            </button>
          ))}
          <button
            onClick={loadEvents}
            className="p-1 rounded-md text-neutral-400 hover:text-white hover:bg-[#202020] transition-colors ml-1"
            title="Aktualisieren"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <div className="divide-y divide-[#1f1f1f]">
        {filtered.length === 0 ? (
          <div className="py-12 text-center text-neutral-500">Keine Ereignisse vorhanden.</div>
        ) : (
          filtered.map((ev) => (
            <div key={ev.id || `${ev.actorId}-${ev.timestamp}`} className="py-3 px-1 hover:bg-white/[0.02] transition-colors flex items-start gap-3">
              <div className="mt-1 shrink-0">
                {ev.severity === 'critical' ? (
                  <span className="w-2 h-2 rounded-full bg-rose-500 inline-block" />
                ) : ev.severity === 'warning' ? (
                  <span className="w-2 h-2 rounded-full bg-amber-500 inline-block" />
                ) : (
                  <span className="w-2 h-2 rounded-full bg-neutral-500 inline-block" />
                )}
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-white text-xs">{ev.actorName}</span>
                  <span className="text-[11px] text-neutral-500 font-mono shrink-0">{formatDate(ev.timestamp)}</span>
                </div>
                <div className="flex items-center gap-2 mt-1">
                  <span className="px-1.5 py-0.5 rounded text-[10px] uppercase font-mono bg-[#1c1c1c] text-neutral-300 border border-[#2a2a2a]">
                    {ev.type}:{ev.action}
                  </span>
                  <span className="text-neutral-500 font-mono text-[11px] truncate max-w-sm">ID: {ev.actorId.slice(0, 12)}</span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

