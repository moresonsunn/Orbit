import React, { useState, useEffect } from 'react';
import { Globe, Trash2, ArrowRight } from 'lucide-react';
import { DockerNetwork, AuthStatus } from '../types';
import { api } from '../lib/api';
import { ActionConfirmModal } from '../components/modals/ActionConfirmModal';

interface NetworksPageProps {
  authStatus?: AuthStatus;
}

export const NetworksPage: React.FC<NetworksPageProps> = ({ authStatus }) => {
  const [networks, setNetworks] = useState<DockerNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedNetwork, setSelectedNetwork] = useState<DockerNetwork | null>(null);
  const [deleteModal, setDeleteModal] = useState<DockerNetwork | null>(null);

  const destructiveEnabled = authStatus?.features.enableDestructive ?? false;

  const loadNetworks = async () => {
    try {
      const data = await api.getNetworks();
      setNetworks(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadNetworks();
  }, []);

  const handleDelete = async () => {
    if (!deleteModal) return;
    try {
      await api.removeNetwork(deleteModal.Id);
      setDeleteModal(null);
      loadNetworks();
    } catch (err: any) {
      alert(`Fehler: ${err.message}`);
    }
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Networks</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Übersicht aller erkannten Docker-Netzwerktreiber und Subnetze.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {networks.map((net) => {
          const subnet = net.IPAM?.Config?.[0]?.Subnet || '—';
          const gateway = net.IPAM?.Config?.[0]?.Gateway || '—';

          return (
            <div
              key={net.Id}
              className="p-4 rounded-lg bg-[#181818] border border-[#282828] hover:border-[#383838] transition-colors space-y-3"
            >
              <div className="flex items-start justify-between">
                <div>
                  <span className="text-sm font-semibold text-white block">{net.Name}</span>
                  <span className="text-[11px] text-neutral-500 font-mono">ID: {net.Id.slice(0, 12)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-[#222222] text-neutral-300 border border-[#2e2e2e]">
                    {net.Driver}
                  </span>
                  {destructiveEnabled && !['bridge', 'host', 'none'].includes(net.Name) && (
                    <button
                      onClick={() => setDeleteModal(net)}
                      className="p-1 rounded text-neutral-500 hover:text-rose-400 hover:bg-[#222222] transition-colors"
                      title="Netzwerk löschen"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              </div>

              <div className="space-y-1.5 font-mono text-xs pt-2 border-t border-[#242424]">
                <div className="flex justify-between text-neutral-400">
                  <span className="font-sans">Subnetz:</span>
                  <span className="text-neutral-200">{subnet}</span>
                </div>
                <div className="flex justify-between text-neutral-400">
                  <span className="font-sans">Gateway:</span>
                  <span className="text-neutral-200">{gateway}</span>
                </div>
                <div className="flex justify-between text-neutral-400">
                  <span className="font-sans">Scope:</span>
                  <span className="text-neutral-300">{net.Scope}</span>
                </div>
                <div className="flex justify-between text-neutral-400">
                  <span className="font-sans">Intern:</span>
                  <span className="text-neutral-300">{net.Internal ? 'Ja' : 'Nein'}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <ActionConfirmModal
        isOpen={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        onConfirm={handleDelete}
        title={`Netzwerk '${deleteModal?.Name}' löschen?`}
        description="Möchtest du dieses Docker-Netzwerk wirklich entfernen?"
        isDestructive={true}
        confirmText="Löschen"
      />
    </div>
  );
};

