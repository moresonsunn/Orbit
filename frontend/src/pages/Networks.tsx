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
    <div className="space-y-6 animate-in fade-in duration-200">
      <div className="flex items-center justify-between p-4 rounded-2xl bg-slate-900/60 border border-slate-800/80">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-indigo-950/60 border border-indigo-800/60 text-indigo-400">
            <Globe className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-white tracking-tight">Docker Networks</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Übersicht aller erkannten Docker-Netzwerktreiber und Subnetze
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {networks.map((net) => {
          const subnet = net.IPAM?.Config?.[0]?.Subnet || '—';
          const gateway = net.IPAM?.Config?.[0]?.Gateway || '—';

          return (
            <div
              key={net.Id}
              className="p-5 rounded-2xl bg-slate-900/60 border border-slate-800/80 hover:border-slate-700/80 transition-all shadow-sm space-y-3"
            >
              <div className="flex items-start justify-between">
                <div>
                  <span className="text-sm font-bold text-white block">{net.Name}</span>
                  <span className="text-[10px] text-slate-400 font-mono">ID: {net.Id.slice(0, 12)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-slate-800 text-cyan-400 border border-slate-700">
                    {net.Driver}
                  </span>
                  {destructiveEnabled && !['bridge', 'host', 'none'].includes(net.Name) && (
                    <button
                      onClick={() => setDeleteModal(net)}
                      className="p-1 rounded text-slate-400 hover:text-rose-400 hover:bg-slate-800 transition-colors"
                      title="Netzwerk löschen"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              </div>

              <div className="space-y-1.5 font-mono text-xs pt-2 border-t border-slate-800/60">
                <div className="flex justify-between text-slate-400">
                  <span>Subnetz:</span>
                  <span className="text-slate-200">{subnet}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Gateway:</span>
                  <span className="text-slate-200">{gateway}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Scope:</span>
                  <span className="text-slate-200">{net.Scope}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Intern:</span>
                  <span className="text-slate-200">{net.Internal ? 'Ja' : 'Nein'}</span>
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

