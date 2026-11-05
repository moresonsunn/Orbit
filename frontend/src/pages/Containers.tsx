import React, { useState } from 'react';
import {
  Search,
  Play,
  Square,
  RotateCw,
  Pause,
  Trash2,
  Terminal,
  ExternalLink,
  CheckSquare,
  Square as UncheckedSquare,
  Layers,
  Filter,
} from 'lucide-react';
import { Container, AuthStatus } from '../types';
import { StatusBadge } from '../components/common/StatusBadge';
import { formatBytes, formatBps, formatPercent, formatUptime } from '../lib/utils';
import { api } from '../lib/api';
import { ActionConfirmModal } from '../components/modals/ActionConfirmModal';

interface ContainersPageProps {
  containers: Container[];
  authStatus?: AuthStatus;
  onSelectContainer: (id: string) => void;
  onRefresh: () => void;
}

export const ContainersPage: React.FC<ContainersPageProps> = ({
  containers,
  authStatus,
  onSelectContainer,
  onRefresh,
}) => {
  const [search, setSearch] = useState('');
  const [filterState, setFilterState] = useState<'all' | 'running' | 'stopped' | 'unhealthy'>('all');
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);
  const [activeModal, setActiveModal] = useState<{
    action: 'start' | 'stop' | 'restart' | 'remove' | 'bulk-start' | 'bulk-stop' | 'bulk-restart';
    targetId?: string;
    targetName?: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);

  const actionsEnabled = authStatus?.features.enableActions ?? false;
  const destructiveEnabled = authStatus?.features.enableDestructive ?? false;

  // Filter containers
  const filtered = containers.filter((c) => {
    const matchesSearch =
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.image.toLowerCase().includes(search.toLowerCase()) ||
      c.shortId.toLowerCase().includes(search.toLowerCase()) ||
      (c.composeProject && c.composeProject.toLowerCase().includes(search.toLowerCase()));

    if (!matchesSearch) return false;

    if (filterState === 'running') return c.state === 'running';
    if (filterState === 'stopped') return c.state !== 'running';
    if (filterState === 'unhealthy') return c.health === 'unhealthy';
    return true;
  });

  const toggleSelectAll = () => {
    if (selectedIDs.length === filtered.length) {
      setSelectedIDs([]);
    } else {
      setSelectedIDs(filtered.map((c) => c.id));
    }
  };

  const toggleSelect = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setSelectedIDs((prev) => (prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]));
  };

  const handleAction = async () => {
    if (!activeModal) return;
    setLoading(true);

    try {
      if (activeModal.action === 'start' && activeModal.targetId) {
        await api.startContainer(activeModal.targetId);
      } else if (activeModal.action === 'stop' && activeModal.targetId) {
        await api.stopContainer(activeModal.targetId);
      } else if (activeModal.action === 'restart' && activeModal.targetId) {
        await api.restartContainer(activeModal.targetId);
      } else if (activeModal.action === 'remove' && activeModal.targetId) {
        await api.removeContainer(activeModal.targetId);
      } else if (activeModal.action === 'bulk-start') {
        await api.bulkStart(selectedIDs);
        setSelectedIDs([]);
      } else if (activeModal.action === 'bulk-stop') {
        await api.bulkStop(selectedIDs);
        setSelectedIDs([]);
      } else if (activeModal.action === 'bulk-restart') {
        await api.bulkRestart(selectedIDs);
        setSelectedIDs([]);
      }
      onRefresh();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    } finally {
      setLoading(false);
      setActiveModal(null);
    }
  };

  return (
    <div className="space-y-4 animate-in fade-in duration-200">
      {/* Controls Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-slate-900/60 p-3 rounded-2xl border border-slate-800/80">
        <div className="flex items-center gap-2 flex-1 max-w-md">
          <div className="relative w-full">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Container nach Name, Image oder ID durchsuchen..."
              className="w-full pl-9 pr-4 py-2 bg-slate-950 border border-slate-800 rounded-xl text-xs text-white placeholder-slate-400 focus:outline-none focus:border-cyan-500"
            />
          </div>
        </div>

        {/* State Filter Tabs */}
        <div className="flex items-center gap-1 bg-slate-950 p-1 rounded-xl border border-slate-800 self-start sm:self-auto">
          {(['all', 'running', 'stopped', 'unhealthy'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setFilterState(tab)}
              className={`px-3 py-1 rounded-lg text-xs font-medium capitalize transition-all ${
                filterState === tab
                  ? 'bg-cyan-950/80 text-cyan-300 border border-cyan-800/60 shadow-sm'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              {tab === 'all'
                ? `Alle (${containers.length})`
                : tab === 'running'
                ? `Läuft (${containers.filter((c) => c.state === 'running').length})`
                : tab === 'stopped'
                ? `Gestoppt (${containers.filter((c) => c.state !== 'running').length})`
                : `Probleme (${containers.filter((c) => c.health === 'unhealthy').length})`}
            </button>
          ))}
        </div>
      </div>

      {/* Bulk Action Bar (visible when items selected) */}
      {selectedIDs.length > 0 && (
        <div className="flex items-center justify-between bg-cyan-950/40 border border-cyan-800/60 p-3 rounded-xl animate-in slide-in-from-top-2 duration-200">
          <span className="text-xs font-semibold text-cyan-300 font-mono">
            {selectedIDs.length} Container ausgewählt
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveModal({ action: 'bulk-start' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600/80 hover:bg-emerald-600 disabled:opacity-40 text-white rounded-lg text-xs font-semibold transition-all shadow-sm"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Starten</span>
            </button>
            <button
              onClick={() => setActiveModal({ action: 'bulk-stop' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-600/80 hover:bg-amber-600 disabled:opacity-40 text-white rounded-lg text-xs font-semibold transition-all shadow-sm"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
              <span>Stoppen</span>
            </button>
            <button
              onClick={() => setActiveModal({ action: 'bulk-restart' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600/80 hover:bg-indigo-600 disabled:opacity-40 text-white rounded-lg text-xs font-semibold transition-all shadow-sm"
            >
              <RotateCw className="w-3.5 h-3.5" />
              <span>Neustarten</span>
            </button>
            <button
              onClick={() => setSelectedIDs([])}
              className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-medium transition-colors"
            >
              Auswahl aufheben
            </button>
          </div>
        </div>
      )}

      {/* Main Container Table */}
      <div className="overflow-hidden rounded-2xl bg-slate-900/60 border border-slate-800/80 shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 uppercase tracking-wider font-mono bg-slate-950/40">
                <th className="py-3 px-4 w-8">
                  <button onClick={toggleSelectAll} className="text-slate-400 hover:text-white">
                    {selectedIDs.length > 0 && selectedIDs.length === filtered.length ? (
                      <CheckSquare className="w-4 h-4 text-cyan-400" />
                    ) : (
                      <UncheckedSquare className="w-4 h-4" />
                    )}
                  </button>
                </th>
                <th className="py-3 px-3 font-medium">Container</th>
                <th className="py-3 px-3 font-medium">Status</th>
                <th className="py-3 px-3 font-medium">CPU</th>
                <th className="py-3 px-3 font-medium">RAM</th>
                <th className="py-3 px-3 font-medium">Netzwerk</th>
                <th className="py-3 px-3 font-medium">Disk I/O</th>
                <th className="py-3 px-3 font-medium">Uptime</th>
                <th className="py-3 px-3 font-medium">Ports</th>
                <th className="py-3 px-4 font-medium text-right">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/50">
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={10} className="py-8 text-center text-slate-400">
                    Keine passenden Container gefunden.
                  </td>
                </tr>
              ) : (
                filtered.map((c) => {
                  const isSelected = selectedIDs.includes(c.id);
                  const isRunning = c.state === 'running';

                  return (
                    <tr
                      key={c.id}
                      onClick={() => onSelectContainer(c.id)}
                      className={`hover:bg-slate-800/40 cursor-pointer transition-colors ${
                        isSelected ? 'bg-cyan-950/20' : ''
                      }`}
                    >
                      <td className="py-3 px-4" onClick={(e) => toggleSelect(c.id, e)}>
                        {isSelected ? (
                          <CheckSquare className="w-4 h-4 text-cyan-400" />
                        ) : (
                          <UncheckedSquare className="w-4 h-4 text-slate-400" />
                        )}
                      </td>
                      <td className="py-3 px-3">
                        <div className="flex flex-col">
                          <span className="font-bold text-white text-sm hover:text-cyan-400 transition-colors">
                            {c.name}
                          </span>
                          <span className="text-[10px] text-slate-400 font-mono truncate max-w-[180px]">
                            {c.imageRepo}:{c.imageTag}
                          </span>
                          {c.composeProject && (
                            <span className="text-[9px] text-indigo-400 font-mono mt-0.5">
                              Compose: {c.composeProject}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-3 px-3">
                        <StatusBadge status={c.state} />
                      </td>
                      <td className="py-3 px-3 font-mono font-medium">
                        {c.stats?.cpuPercent.value !== null && c.stats?.cpuPercent.value !== undefined ? (
                          <span className={c.stats.cpuPercent.value > 80 ? 'text-rose-400 font-bold' : 'text-slate-200'}>
                            {formatPercent(c.stats.cpuPercent.value)}
                          </span>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="py-3 px-3 font-mono">
                        {c.stats?.memUsage.value !== null && c.stats?.memUsage.value !== undefined ? (
                          <div className="flex flex-col">
                            <span className="text-slate-200 font-medium">{formatBytes(c.stats.memUsage.value)}</span>
                            <span className="text-[10px] text-slate-400">{formatPercent(c.stats.memPercent.value)}</span>
                          </div>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="py-3 px-3 font-mono text-[11px] text-slate-300">
                        {c.stats ? (
                          <div className="flex flex-col">
                            <span>↓ {formatBps(c.stats.netRxBps.value)}</span>
                            <span className="text-slate-400">↑ {formatBps(c.stats.netTxBps.value)}</span>
                          </div>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="py-3 px-3 font-mono text-[11px] text-slate-300">
                        {c.stats?.blkReadBps.value !== null && c.stats?.blkReadBps.value !== undefined ? (
                          <div className="flex flex-col">
                            <span>R: {formatBps(c.stats.blkReadBps.value)}</span>
                            <span className="text-slate-400">W: {formatBps(c.stats.blkWriteBps.value)}</span>
                          </div>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="py-3 px-3 font-mono text-slate-400">
                        {c.startedAt
                          ? formatUptime(Math.floor((Date.now() - new Date(c.startedAt).getTime()) / 1000))
                          : '—'}
                      </td>
                      <td className="py-3 px-3 font-mono text-xs">
                        {c.ports.length === 0 ? (
                          <span className="text-slate-400">—</span>
                        ) : (
                          <div className="max-w-[140px] truncate" title={c.ports.map((p) => `${p.publicPort || ''}:${p.privatePort}/${p.protocol}`).join(', ')}>
                            {c.ports.slice(0, 2).map((p, i) => (
                              <span key={i} className="inline-block bg-slate-800 px-1.5 py-0.5 rounded text-[10px] mr-1 text-slate-300">
                                {p.publicPort ? `${p.publicPort}:` : ''}{p.privatePort}
                              </span>
                            ))}
                            {c.ports.length > 2 && <span className="text-slate-400 text-[10px]">+{c.ports.length - 2}</span>}
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-1">
                          {isRunning ? (
                            <>
                              <button
                                onClick={() => setActiveModal({ action: 'restart', targetId: c.id, targetName: c.name })}
                                disabled={!actionsEnabled}
                                className="p-1.5 rounded-lg text-slate-400 hover:text-indigo-400 hover:bg-slate-800 disabled:opacity-30 transition-colors"
                                title="Neustarten"
                              >
                                <RotateCw className="w-3.5 h-3.5" />
                              </button>
                              <button
                                onClick={() => setActiveModal({ action: 'stop', targetId: c.id, targetName: c.name })}
                                disabled={!actionsEnabled}
                                className="p-1.5 rounded-lg text-slate-400 hover:text-amber-400 hover:bg-slate-800 disabled:opacity-30 transition-colors"
                                title="Stoppen"
                              >
                                <Square className="w-3.5 h-3.5" />
                              </button>
                            </>
                          ) : (
                            <button
                              onClick={() => setActiveModal({ action: 'start', targetId: c.id, targetName: c.name })}
                              disabled={!actionsEnabled}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-emerald-400 hover:bg-slate-800 disabled:opacity-30 transition-colors"
                              title="Starten"
                            >
                              <Play className="w-3.5 h-3.5 fill-current" />
                            </button>
                          )}
                          <button
                            onClick={() => onSelectContainer(c.id)}
                            className="p-1.5 rounded-lg text-slate-400 hover:text-cyan-400 hover:bg-slate-800 transition-colors"
                            title="Details öffnen"
                          >
                            <ExternalLink className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => setActiveModal({ action: 'remove', targetId: c.id, targetName: c.name })}
                            disabled={!destructiveEnabled}
                            className="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-800 disabled:opacity-30 transition-colors"
                            title="Container löschen"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Confirmation Modal */}
      <ActionConfirmModal
        isOpen={activeModal !== null}
        onClose={() => setActiveModal(null)}
        onConfirm={handleAction}
        isLoading={loading}
        title={
          activeModal?.action === 'remove'
            ? `Container '${activeModal.targetName}' löschen?`
            : activeModal?.action.startsWith('bulk')
            ? `Massenaktion (${selectedIDs.length} Container)`
            : `Aktion ausführen: ${activeModal?.action}`
        }
        description={
          activeModal?.action === 'remove'
            ? 'Möchtest du diesen Container wirklich entfernen? Diese Aktion kann nicht rückgängig gemacht werden.'
            : activeModal?.action.startsWith('bulk')
            ? `Möchtest du die Aktion wirklich auf alle ausgewählten Container anwenden?`
            : `Soll der Container '${activeModal?.targetName}' ${activeModal?.action} werden?`
        }
        affectedItems={activeModal?.action.startsWith('bulk') ? selectedIDs : undefined}
        isDestructive={activeModal?.action === 'remove'}
        confirmText={
          activeModal?.action === 'remove'
            ? 'Löschen'
            : activeModal?.action === 'stop'
            ? 'Stoppen'
            : 'Ausführen'
        }
      />
    </div>
  );
};
