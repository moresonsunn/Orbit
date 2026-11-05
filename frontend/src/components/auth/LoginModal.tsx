import React, { useState } from 'react';
import { Lock, User, AlertCircle, LogIn } from 'lucide-react';
import { api, setCSRFToken } from '../../lib/api';

interface LoginModalProps {
  isOpen: boolean;
  onSuccess: () => void;
}

export const LoginModal: React.FC<LoginModalProps> = ({ isOpen, onSuccess }) => {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const res = await api.login({ username: username.trim(), password });
      setCSRFToken(res.csrfToken);
      onSuccess();
    } catch (err: any) {
      setError(err.message || 'Anmeldung fehlgeschlagen');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xs font-sans">
      <div className="w-full max-w-sm bg-[#181818] border border-[#282828] rounded-xl shadow-2xl p-7">
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-9 h-9 rounded-md bg-blue-600 text-white font-semibold text-sm mb-3">
            O
          </div>
          <h2 className="text-base font-semibold text-white tracking-tight">Bei Orbit anmelden</h2>
          <p className="text-xs text-neutral-400 mt-1 font-normal">Authentifizierung für Verwaltung & Monitoring</p>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-md bg-rose-950/30 border border-rose-800/50 text-rose-300 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">Benutzername</label>
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
            <label className="block text-xs font-medium text-neutral-300 mb-1.5">Passwort</label>
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

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-md text-xs font-medium transition-colors mt-2 flex items-center justify-center gap-2"
          >
            <LogIn className="w-3.5 h-3.5" />
            {loading ? 'Anmelden...' : 'Anmelden'}
          </button>
        </form>
      </div>
    </div>
  );
};

