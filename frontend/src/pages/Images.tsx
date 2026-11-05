import React, { useState, useEffect } from 'react';
import { Layers, Trash2, Scissors } from 'lucide-react';
import { DockerImage, AuthStatus } from '../types';
import { api } from '../lib/api';
import { formatBytes, formatDate } from '../lib/utils';
import { ActionConfirmModal } from '../components/modals/ActionConfirmModal';

interface ImagesPageProps {
  authStatus?: AuthStatus;
}

export const ImagesPage: React.FC<ImagesPageProps> = ({ authStatus }) => {
  const [images, setImages] = useState<DockerImage[]>([]);
  const [loading, setLoading] = useState(true);
  const [pruneModal, setPruneModal] = useState<boolean>(false);
  const [deleteModal, setDeleteModal] = useState<DockerImage | null>(null);

  const destructiveEnabled = authStatus?.features.enableDestructive ?? false;

  const loadImages = async () => {
    try {
      const data = await api.getImages();
      setImages(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadImages();
  }, []);

  const handleDelete = async () => {
    if (!deleteModal) return;
    try {
      await api.removeImage(deleteModal.Id);
      setDeleteModal(null);
      loadImages();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    }
  };

  const handlePrune = async () => {
    try {
      await api.pruneImages(false);
      setPruneModal(false);
      loadImages();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-lg bg-[#151b26] border border-[#232d3f]">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-[#0e121b] border border-[#232d3f] text-blue-400">
            <Layers className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100 tracking-tight">Docker Images ({images.length})</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Lokale Image-Registry und belegter Speicherplatz
            </p>
          </div>
        </div>

        {destructiveEnabled && (
          <button
            onClick={() => setPruneModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#0e121b] border border-[#232d3f] hover:bg-[#1c2433] text-xs font-medium text-slate-200 transition-colors"
          >
            <Scissors className="w-3.5 h-3.5 text-amber-400" />
            <span>Ungenutzte Images bereinigen</span>
          </button>
        )}
      </div>

      <div className="overflow-hidden rounded-lg bg-[#151b26] border border-[#232d3f] shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-[#232d3f] text-slate-400 uppercase tracking-wider bg-[#0e121b]/60">
                <th className="py-2.5 px-4 font-medium">Repository / Tag</th>
                <th className="py-2.5 px-3 font-medium">Image ID</th>
                <th className="py-2.5 px-3 font-medium">Größe</th>
                <th className="py-2.5 px-3 font-medium">Erstellt</th>
                <th className="py-2.5 px-3 font-medium">Container</th>
                <th className="py-2.5 px-4 font-medium text-right">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#232d3f]/60">
              {images.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-slate-500">
                    Keine Images gefunden.
                  </td>
                </tr>
              ) : (
                images.map((img) => {
                  const tag = img.RepoTags && img.RepoTags.length > 0 ? img.RepoTags[0] : '<none>:<none>';
                  const shortId = img.Id.replace('sha256:', '').slice(0, 12);

                  return (
                    <tr key={img.Id} className="hover:bg-[#1a2230] transition-colors">
                      <td className="py-3 px-4 font-semibold text-slate-100 text-xs font-mono">{tag}</td>
                      <td className="py-3 px-3 text-slate-400">{shortId}</td>
                      <td className="py-3 px-3 text-slate-200">{formatBytes(img.Size)}</td>
                      <td className="py-3 px-3 text-slate-400">
                        {img.Created ? formatDate(new Date(img.Created * 1000).toISOString()) : '—'}
                      </td>
                      <td className="py-3 px-3">
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-medium border ${
                            img.Containers > 0
                              ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/60'
                              : 'bg-[#0e121b] text-slate-400 border-[#232d3f]'
                          }`}
                        >
                          {img.Containers} verwendet
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right">
                        {destructiveEnabled && (
                          <button
                            onClick={() => setDeleteModal(img)}
                            className="p-1 rounded-md text-slate-400 hover:text-rose-400 hover:bg-[#1c2433] transition-colors"
                            title="Image löschen"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      <ActionConfirmModal
        isOpen={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        onConfirm={handleDelete}
        title={`Image löschen?`}
        description={`Soll das Image '${deleteModal?.RepoTags?.[0] || deleteModal?.Id}' wirklich entfernt werden?`}
        isDestructive={true}
        confirmText="Löschen"
      />

      <ActionConfirmModal
        isOpen={pruneModal}
        onClose={() => setPruneModal(false)}
        onConfirm={handlePrune}
        title="Ungenutzte Images bereinigen?"
        description="Alle ungenutzten / dangling Images werden vom Server entfernt, um Speicherplatz freizugeben."
        isDestructive={true}
        confirmText="Bereinigen"
      />
    </div>
  );
};

