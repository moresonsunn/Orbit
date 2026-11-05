import React, { useState } from 'react';
import { ShieldCheck, Key, Lock, User, AlertCircle } from 'lucide-react';
import { api, setCSRFToken } from '../../lib/api';

interface SetupModalProps {
  isOpen: boolean;
  onSuccess: () => void;
}

export const SetupModal: React.FC<SetupModalProps> = ({ isOpen, onSuccess }) => {
  const [token, setToken] = useState('');
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      setError('Passwörter stimmen nicht überein');
      return;
    }
    if (password.length < 8) {
      setError('Passwort muss mindestens 8 Zeichen lang sein');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await api.setup({ token: token.trim(), username: username.trim(), password });
      if (res.csrfToken) {
        setCSRFToken(res.csrfToken);
      }
      onSuccess();
    } catch (err: any) {
      setError(err.message || 'Setup fehlgeschlagen');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/85 backdrop-blur-md">
      <div className="w-full max-w-md bg-[#151b26] border border-[#232d3f] rounded-lg shadow-2xl p-6 overflow-hidden">
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-[#0e121b] border border-[#232d3f] text-blue-400 mb-3">
            <ShieldCheck className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-semibold text-slate-100 tracking-tight">Orbit Ersteinrichtung</h2>
          <p className="text-xs text-slate-400 mt-1 font-normal">
            Kein Administrator vorhanden. Bitte gib den im Container-Log angezeigten Setup-Token ein.
          </p>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-md bg-rose-950/40 border border-rose-800/60 text-rose-300 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">
              Einmaliger Setup-Token
            </label>
            <div className="relative">
              <Key className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
              <input
                type="text"
                required
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Aus Container-Log kopieren"
                className="w-full pl-9 pr-3 py-2 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs font-mono text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">Administrator-Benutzername</label>
            <div className="relative">
              <User className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs text-slate-200 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">Neues Passwort (min. 8 Zeichen)</label>
            <div className="relative">
              <Lock className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs text-slate-200 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">Passwort bestätigen</label>
            <div className="relative">
              <Lock className="w-4 h-4 absolute left-3 top-2.5 text-slate-500" />
              <input
                type="password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs text-slate-200 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/20"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-md text-xs font-medium transition-all shadow-xs mt-2"
          >
            {loading ? 'Konto wird erstellt...' : 'Admin-Konto anlegen'}
          </button>
        </form>
      </div>
    </div>
  );
};

