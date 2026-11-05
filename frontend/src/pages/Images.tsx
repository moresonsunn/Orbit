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
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Images</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Lokale Image-Registry und belegter Speicherplatz ({images.length} Images).
          </p>
        </div>

        {destructiveEnabled && (
          <button
            onClick={() => setPruneModal(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#1c1c1c] border border-[#2a2a2a] hover:bg-[#252525] text-xs font-medium text-neutral-300 transition-colors self-start sm:self-auto"
          >
            <Scissors className="w-3.5 h-3.5 text-neutral-400" />
            <span>Ungenutzte Images bereinigen</span>
          </button>
        )}
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead>
            <tr className="border-b border-[#242424] text-neutral-400 uppercase tracking-wider text-[11px] font-semibold font-sans">
              <th className="pb-3 px-3 font-semibold">Repository / Tag</th>
              <th className="pb-3 px-3 font-semibold">Image ID</th>
              <th className="pb-3 px-3 font-semibold">Größe</th>
              <th className="pb-3 px-3 font-semibold">Erstellt</th>
              <th className="pb-3 px-3 font-semibold">Container</th>
              <th className="pb-3 px-3 font-semibold text-right">Aktionen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#1f1f1f]">
            {images.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-8 text-center text-neutral-500 font-sans">
                  Keine Images gefunden.
                </td>
              </tr>
            ) : (
              images.map((img) => {
                const tag = img.RepoTags && img.RepoTags.length > 0 ? img.RepoTags[0] : '<none>:<none>';
                const shortId = img.Id.replace('sha256:', '').slice(0, 12);

                return (
                  <tr key={img.Id} className="hover:bg-white/[0.02] transition-colors">
                    <td className="py-3 px-3 font-semibold text-white text-xs">{tag}</td>
                    <td className="py-3 px-3 text-neutral-400">{shortId}</td>
                    <td className="py-3 px-3 text-neutral-200">{formatBytes(img.Size)}</td>
                    <td className="py-3 px-3 text-neutral-400">
                      {img.Created ? formatDate(new Date(img.Created * 1000).toISOString()) : '—'}
                    </td>
                    <td className="py-3 px-3">
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] font-normal ${
                          img.Containers > 0
                            ? 'bg-[#1e1e1e] text-neutral-200 border border-[#2e2e2e]'
                            : 'bg-transparent text-neutral-500'
                        }`}
                      >
                        {img.Containers} verwendet
                      </span>
                    </td>
                    <td className="py-3 px-3 text-right">
                      {destructiveEnabled && (
                        <button
                          onClick={() => setDeleteModal(img)}
                          className="p-1 rounded text-neutral-500 hover:text-rose-400 hover:bg-[#222222] transition-colors"
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

