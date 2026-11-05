import React, { useState, useEffect } from 'react';
import { HardDrive, Trash2, Scissors } from 'lucide-react';
import { DockerVolume, AuthStatus } from '../types';
import { api } from '../lib/api';
import { formatBytes, formatDate } from '../lib/utils';
import { ActionConfirmModal } from '../components/modals/ActionConfirmModal';

interface VolumesPageProps {
  authStatus?: AuthStatus;
}

export const VolumesPage: React.FC<VolumesPageProps> = ({ authStatus }) => {
  const [volumes, setVolumes] = useState<DockerVolume[]>([]);
  const [loading, setLoading] = useState(true);
  const [pruneModal, setPruneModal] = useState<boolean>(false);
  const [deleteModal, setDeleteModal] = useState<DockerVolume | null>(null);

  const destructiveEnabled = authStatus?.features.enableDestructive ?? false;

  const loadVolumes = async () => {
    try {
      const data = await api.getVolumes();
      setVolumes(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVolumes();
  }, []);

  const handleDelete = async () => {
    if (!deleteModal) return;
    try {
      await api.removeVolume(deleteModal.name);
      setDeleteModal(null);
      loadVolumes();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    }
  };

  const handlePrune = async () => {
    try {
      await api.pruneVolumes(false);
      setPruneModal(false);
      loadVolumes();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-lg bg-[#151b26] border border-[#232d3f]">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-[#0e121b] border border-[#232d3f] text-amber-400">
            <HardDrive className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100 tracking-tight">Docker Volumes ({volumes.length})</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Persistente Speicher-Volumes und Mountpoints
            </p>
          </div>
        </div>

        {destructiveEnabled && (
          <button
            onClick={() => setPruneModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#0e121b] border border-[#232d3f] hover:bg-[#1c2433] text-xs font-medium text-slate-200 transition-colors"
          >
            <Scissors className="w-3.5 h-3.5 text-amber-400" />
            <span>Ungenutzte Volumes bereinigen</span>
          </button>
        )}
      </div>

      <div className="overflow-hidden rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-[#232d3f] text-slate-400 uppercase tracking-wider bg-[#0e121b]/60">
                <th className="py-2.5 px-4 font-medium">Volume Name</th>
                <th className="py-2.5 px-3 font-medium">Treiber</th>
                <th className="py-2.5 px-3 font-medium">Mountpoint</th>
                <th className="py-2.5 px-3 font-medium">Belegter Speicher</th>
                <th className="py-2.5 px-3 font-medium">Erstellt</th>
                <th className="py-2.5 px-4 font-medium text-right">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#232d3f]/60">
              {volumes.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-slate-500">
                    Keine Docker Volumes vorhanden.
                  </td>
                </tr>
              ) : (
                volumes.map((vol) => (
                  <tr key={vol.name} className="hover:bg-[#1a2230] transition-colors">
                    <td className="py-3 px-4 font-semibold text-slate-100 text-xs truncate max-w-[200px]" title={vol.name}>
                      {vol.name}
                    </td>
                    <td className="py-3 px-3 text-blue-400">{vol.driver}</td>
                    <td className="py-3 px-3 text-slate-400 truncate max-w-[240px]" title={vol.mountpoint}>
                      {vol.mountpoint}
                    </td>
                    <td className="py-3 px-3 text-slate-200">
                      {vol.sizeBytes !== undefined ? formatBytes(vol.sizeBytes) : '—'}
                    </td>
                    <td className="py-3 px-3 text-slate-400">{formatDate(vol.createdAt)}</td>
                    <td className="py-3 px-4 text-right">
                      {destructiveEnabled && (
                        <button
                          onClick={() => setDeleteModal(vol)}
                          className="p-1 rounded-md text-slate-400 hover:text-rose-400 hover:bg-[#1c2433] transition-colors"
                          title="Volume löschen"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <ActionConfirmModal
        isOpen={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        onConfirm={handleDelete}
        title={`Volume löschen?`}
        description={`Soll das Volume '${deleteModal?.name}' wirklich gelöscht werden? Falls es von laufenden Containern verwendet wird, bricht Orbit den Vorgang ab.`}
        isDestructive={true}
        confirmText="Löschen"
      />

      <ActionConfirmModal
        isOpen={pruneModal}
        onClose={() => setPruneModal(false)}
        onConfirm={handlePrune}
        title="Ungenutzte Volumes bereinigen?"
        description="Alle ungenutzten Docker-Volumes werden unwiderruflich gelöscht."
        isDestructive={true}
        confirmText="Bereinigen"
      />
    </div>
  );
};

