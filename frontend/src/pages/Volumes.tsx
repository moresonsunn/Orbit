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
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Volumes</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Persistente Speicher-Volumes und Mountpoints ({volumes.length} Volumes).
          </p>
        </div>

        {destructiveEnabled && (
          <button
            onClick={() => setPruneModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#1c1c1c] border border-[#2a2a2a] hover:bg-[#252525] text-xs font-medium text-neutral-300 transition-colors self-start sm:self-auto"
          >
            <Scissors className="w-3.5 h-3.5 text-neutral-400" />
            <span>Ungenutzte Volumes bereinigen</span>
          </button>
        )}
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold font-sans">
              <th className="pb-3 px-3 font-semibold">Volume Name</th>
              <th className="pb-3 px-3 font-semibold">Treiber</th>
              <th className="pb-3 px-3 font-semibold">Mountpoint</th>
              <th className="pb-3 px-3 font-semibold">Belegter Speicher</th>
              <th className="pb-3 px-3 font-semibold">Erstellt</th>
              <th className="pb-3 px-3 font-semibold text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#1f1f1f]">
            {volumes.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-8 text-center text-neutral-500 font-sans">
                  Keine Docker Volumes vorhanden.
                </td>
              </tr>
            ) : (
              volumes.map((vol) => (
                <tr key={vol.name} className="hover:bg-white/[0.02] transition-colors">
                  <td className="py-3 px-3 font-semibold text-white text-xs truncate max-w-[200px]" title={vol.name}>
                    {vol.name}
                  </td>
                  <td className="py-3 px-3 text-neutral-300">{vol.driver}</td>
                  <td className="py-3 px-3 text-neutral-400 truncate max-w-[260px]" title={vol.mountpoint}>
                    {vol.mountpoint}
                  </td>
                  <td className="py-3 px-3 text-neutral-200">
                    {vol.sizeBytes !== undefined ? formatBytes(vol.sizeBytes) : '—'}
                  </td>
                  <td className="py-3 px-3 text-neutral-400">{formatDate(vol.createdAt)}</td>
                  <td className="py-3 px-3 text-right">
                    {destructiveEnabled && (
                      <button
                        onClick={() => setDeleteModal(vol)}
                        className="p-1 rounded text-neutral-500 hover:text-rose-400 hover:bg-[#222222] transition-colors"
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

