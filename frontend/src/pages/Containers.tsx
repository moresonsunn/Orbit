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
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Page Title & Subtitle (Tailscale Machines header style) */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Containers</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Verwalte und überwache alle Container auf der lokalen Docker Engine.
          </p>
        </div>
      </div>

      {/* Tailscale Filter Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2">
        <div className="flex flex-wrap items-center gap-2 flex-1">
          <div className="relative min-w-[240px] max-w-sm">
            <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by name, image, or ID..."
              className="w-full pl-8 pr-3 py-1.5 bg-[#181818] border border-[#2a2a2a] rounded-md text-xs text-white placeholder-neutral-500 focus:outline-none focus:border-neutral-400"
            />
          </div>

          {/* Filter Pills */}
          <div className="flex items-center gap-1">
            {(['all', 'running', 'stopped', 'unhealthy'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setFilterState(tab)}
                className={`px-2.5 py-1 rounded-md text-xs font-medium capitalize transition-colors ${
                  filterState === tab
                    ? 'bg-white/10 text-white'
                    : 'text-neutral-400 hover:text-white hover:bg-white/[0.04]'
                }`}
              >
                {tab === 'all'
                  ? `Alle`
                  : tab === 'running'
                  ? `Läuft`
                  : tab === 'stopped'
                  ? `Gestoppt`
                  : `Problem`}
              </button>
            ))}
          </div>

          <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#202020] text-neutral-400 font-medium">
            {filtered.length} {filtered.length === 1 ? 'container' : 'containers'}
          </span>
        </div>
      </div>

      {/* Bulk Action Bar (visible when items selected) */}
      {selectedIDs.length > 0 && (
        <div className="flex items-center justify-between bg-[#1a1a1a] border border-blue-500/40 p-3 rounded-lg shadow-sm animate-in slide-in-from-top-2 duration-200">
          <span className="text-xs font-medium text-neutral-200">
            {selectedIDs.length} Container ausgewählt
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveModal({ action: 'bulk-start' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-40 text-white rounded-md text-xs font-medium transition-colors shadow-xs"
            >
              <Play className="w-3 h-3 fill-current" />
              <span>Starten</span>
            </button>
            <button
              onClick={() => setActiveModal({ action: 'bulk-stop' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-40 text-neutral-200 rounded-md text-xs font-medium transition-colors"
            >
              <Square className="w-3 h-3 fill-current" />
              <span>Stoppen</span>
            </button>
            <button
              onClick={() => setActiveModal({ action: 'bulk-restart' })}
              disabled={!actionsEnabled}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-40 text-neutral-200 rounded-md text-xs font-medium transition-colors"
            >
              <RotateCw className="w-3 h-3" />
              <span>Neustarten</span>
            </button>
            <button
              onClick={() => setSelectedIDs([])}
              className="px-3 py-1.5 text-neutral-400 hover:text-white text-xs transition-colors"
            >
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {/* Main Container Table (Direct on Canvas, Tailscale style) */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead>
            <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold">
              <th className="pb-3 px-2 w-6">
                <button onClick={toggleSelectAll} className="text-neutral-500 hover:text-white">
                  {selectedIDs.length > 0 && selectedIDs.length === filtered.length ? (
                    <CheckSquare className="w-4 h-4 text-blue-500" />
                  ) : (
                    <UncheckedSquare className="w-4 h-4" />
                  )}
                </button>
              </th>
              <th className="pb-3 px-3 font-semibold">Container</th>
              <th className="pb-3 px-3 font-semibold">Ports</th>
              <th className="pb-3 px-3 font-semibold">CPU</th>
              <th className="pb-3 px-3 font-semibold">RAM</th>
              <th className="pb-3 px-3 font-semibold">Netzwerk</th>
              <th className="pb-3 px-3 font-semibold">Uptime</th>
              <th className="pb-3 px-3 font-semibold">Status</th>
              <th className="pb-3 px-3 font-semibold text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#1f1f1f]">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={9} className="py-12 text-center text-neutral-500">
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
                    className={`hover:bg-white/[0.02] cursor-pointer transition-colors ${
                      isSelected ? 'bg-blue-950/20' : ''
                    }`}
                  >
                    <td className="py-3 px-2" onClick={(e) => toggleSelect(c.id, e)}>
                      {isSelected ? (
                        <CheckSquare className="w-4 h-4 text-blue-500" />
                      ) : (
                        <UncheckedSquare className="w-4 h-4 text-neutral-600 hover:text-neutral-400" />
                      )}
                    </td>
                    <td className="py-3 px-3">
                      <div className="flex flex-col">
                        <span className="font-semibold text-white text-[13px] hover:underline">
                          {c.name}
                        </span>
                        <span className="text-xs text-neutral-400 font-mono truncate max-w-[220px]">
                          {c.imageRepo}:{c.imageTag}
                        </span>
                        {c.composeProject && (
                          <span className="inline-block self-start mt-0.5 px-1.5 py-0.5 rounded text-[10px] bg-[#1e1e1e] text-neutral-400 font-mono">
                            compose: {c.composeProject}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="py-3 px-3 font-mono text-xs">
                      {c.ports.length === 0 ? (
                        <span className="text-neutral-600">—</span>
                      ) : (
                        <div className="max-w-[160px] truncate" title={c.ports.map((p) => `${p.publicPort || ''}:${p.privatePort}/${p.protocol}`).join(', ')}>
                          {c.ports.slice(0, 2).map((p, i) => (
                            <span key={i} className="inline-block bg-[#1a1a1a] border border-[#262626] px-1.5 py-0.5 rounded text-[11px] mr-1 text-neutral-300">
                              {p.publicPort ? `${p.publicPort}:` : ''}{p.privatePort}
                            </span>
                          ))}
                          {c.ports.length > 2 && <span className="text-neutral-500 text-[10px]">+{c.ports.length - 2}</span>}
                        </div>
                      )}
                    </td>
                    <td className="py-3 px-3 font-mono text-xs text-neutral-300">
                      {c.stats?.cpuPercent.value !== null && c.stats?.cpuPercent.value !== undefined ? (
                        <span className={c.stats.cpuPercent.value > 80 ? 'text-rose-400 font-bold' : ''}>
                          {formatPercent(c.stats.cpuPercent.value)}
                        </span>
                      ) : (
                        <span className="text-neutral-600">—</span>
                      )}
                    </td>
                    <td className="py-3 px-3 font-mono text-xs text-neutral-300">
                      {c.stats?.memUsage.value !== null && c.stats?.memUsage.value !== undefined ? (
                        <div className="flex flex-col">
                          <span>{formatBytes(c.stats.memUsage.value)}</span>
                          <span className="text-[10px] text-neutral-500">{formatPercent(c.stats.memPercent.value)}</span>
                        </div>
                      ) : (
                        <span className="text-neutral-600">—</span>
                      )}
                    </td>
                    <td className="py-3 px-3 font-mono text-xs text-neutral-400">
                      {c.stats ? (
                        <div className="flex flex-col text-[11px]">
                          <span>↓ {formatBps(c.stats.netRxBps.value)}</span>
                          <span className="text-neutral-500">↑ {formatBps(c.stats.netTxBps.value)}</span>
                        </div>
                      ) : (
                        <span className="text-neutral-600">—</span>
                      )}
                    </td>
                    <td className="py-3 px-3 font-mono text-xs text-neutral-400">
                      {c.startedAt
                        ? formatUptime(Math.floor((Date.now() - new Date(c.startedAt).getTime()) / 1000))
                        : '—'}
                    </td>
                    <td className="py-3 px-3">
                      <StatusBadge status={c.state} />
                    </td>
                    <td className="py-3 px-3 text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="flex items-center justify-end gap-1">
                        {isRunning ? (
                          <>
                            <button
                              onClick={() => setActiveModal({ action: 'restart', targetId: c.id, targetName: c.name })}
                              disabled={!actionsEnabled}
                              className="p-1.5 rounded text-neutral-400 hover:text-white hover:bg-[#202020] disabled:opacity-30 transition-colors"
                              title="Neustarten"
                            >
                              <RotateCw className="w-3.5 h-3.5" />
                            </button>
                            <button
                              onClick={() => setActiveModal({ action: 'stop', targetId: c.id, targetName: c.name })}
                              disabled={!actionsEnabled}
                              className="p-1.5 rounded text-neutral-400 hover:text-amber-400 hover:bg-[#202020] disabled:opacity-30 transition-colors"
                              title="Stoppen"
                            >
                              <Square className="w-3.5 h-3.5" />
                            </button>
                          </>
                        ) : (
                          <button
                            onClick={() => setActiveModal({ action: 'start', targetId: c.id, targetName: c.name })}
                            disabled={!actionsEnabled}
                            className="p-1.5 rounded text-neutral-400 hover:text-emerald-400 hover:bg-[#202020] disabled:opacity-30 transition-colors"
                            title="Starten"
                          >
                            <Play className="w-3.5 h-3.5 fill-current" />
                          </button>
                        )}
                        <button
                          onClick={() => onSelectContainer(c.id)}
                          className="p-1.5 rounded text-neutral-400 hover:text-white hover:bg-[#202020] transition-colors"
                          title="Details öffnen"
                        >
                          <ExternalLink className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => setActiveModal({ action: 'remove', targetId: c.id, targetName: c.name })}
                          disabled={!destructiveEnabled}
                          className="p-1.5 rounded text-neutral-400 hover:text-rose-400 hover:bg-[#202020] disabled:opacity-30 transition-colors"
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

