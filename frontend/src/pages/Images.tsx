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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-2xl bg-slate-900/60 border border-slate-800/80">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-cyan-950/60 border border-cyan-800/60 text-cyan-400">
            <Layers className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-white tracking-tight">Docker Images ({images.length})</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Lokale Image-Registry und belegter Speicherplatz
            </p>
          </div>
        </div>

        {destructiveEnabled && (
          <button
            onClick={() => setPruneModal(true)}
            className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-slate-200 transition-colors"
          >
            <Scissors className="w-4 h-4 text-amber-400" />
            <span>Ungenutzte Images bereinigen</span>
          </button>
        )}
      </div>

      <div className="overflow-hidden rounded-2xl bg-slate-900/60 border border-slate-800/80 shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 uppercase tracking-wider bg-slate-950/40">
                <th className="py-3 px-4 font-medium">Repository / Tag</th>
                <th className="py-3 px-3 font-medium">Image ID</th>
                <th className="py-3 px-3 font-medium">Größe</th>
                <th className="py-3 px-3 font-medium">Erstellt</th>
                <th className="py-3 px-3 font-medium">Container</th>
                <th className="py-3 px-4 font-medium text-right">Aktionen</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/50">
              {images.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-slate-400">
                    Keine Images gefunden.
                  </td>
                </tr>
              ) : (
                images.map((img) => {
                  const tag = img.RepoTags && img.RepoTags.length > 0 ? img.RepoTags[0] : '<none>:<none>';
                  const shortId = img.Id.replace('sha256:', '').slice(0, 12);

                  return (
                    <tr key={img.Id} className="hover:bg-slate-800/40 transition-colors">
                      <td className="py-3 px-4 font-bold text-white text-sm font-sans">{tag}</td>
                      <td className="py-3 px-3 text-slate-400">{shortId}</td>
                      <td className="py-3 px-3 text-slate-200">{formatBytes(img.Size)}</td>
                      <td className="py-3 px-3 text-slate-400">
                        {img.Created ? formatDate(new Date(img.Created * 1000).toISOString()) : '—'}
                      </td>
                      <td className="py-3 px-3">
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            img.Containers > 0 ? 'bg-emerald-950 text-emerald-300' : 'bg-slate-800 text-slate-400'
                          }`}
                        >
                          {img.Containers} verwendet
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right">
                        {destructiveEnabled && (
                          <button
                            onClick={() => setDeleteModal(img)}
                            className="p-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-800 transition-colors"
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
