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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs font-sans">
      <div className="w-full max-w-md bg-[#181818] border border-[#282828] rounded-xl shadow-2xl p-7">
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-white/[0.05] border border-[#282828] text-neutral-200 mb-3">
            <ShieldCheck className="w-5 h-5 text-blue-500" />
          </div>
          <h2 className="text-base font-semibold text-white tracking-tight">Orbit Ersteinrichtung</h2>
          <p className="text-xs text-neutral-400 mt-1 font-normal">
            Kein Administrator vorhanden. Bitte gib den im Container-Log angezeigten Setup-Token ein.
          </p>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-md bg-rose-950/30 border border-rose-800/50 text-rose-300 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">
              Einmaliger Setup-Token
            </label>
            <div className="relative">
              <Key className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
              <input
                type="text"
                required
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Aus Container-Log kopieren"
                className="w-full pl-9 pr-3 py-2 bg-[#121212] border border-[#282828] rounded-md text-xs font-mono text-neutral-100 placeholder-neutral-600 focus:outline-none focus:border-neutral-500 transition-colors"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">Administrator-Benutzername</label>
            <div className="relative">
              <User className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#121212] border border-[#282828] rounded-md text-xs text-neutral-100 focus:outline-none focus:border-neutral-500 transition-colors"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">Neues Passwort (min. 8 Zeichen)</label>
            <div className="relative">
              <Lock className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#121212] border border-[#282828] rounded-md text-xs text-neutral-100 focus:outline-none focus:border-neutral-500 transition-colors"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">Passwort bestätigen</label>
            <div className="relative">
              <Lock className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
              <input
                type="password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full pl-9 pr-3 py-2 bg-[#121212] border border-[#282828] rounded-md text-xs text-neutral-100 focus:outline-none focus:border-neutral-500 transition-colors"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-md text-xs font-medium transition-colors mt-2"
          >
            {loading ? 'Konto wird erstellt...' : 'Admin-Konto anlegen'}
          </button>
        </form>
      </div>
    </div>
  );
};

