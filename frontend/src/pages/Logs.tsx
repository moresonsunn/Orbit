import React, { useState, useEffect } from 'react';
import { Terminal, Download, Search, RefreshCw } from 'lucide-react';
import { Container } from '../types';
import { api } from '../lib/api';

interface LogsPageProps {
  containers: Container[];
}

export const LogsPage: React.FC<LogsPageProps> = ({ containers }) => {
  const [selectedId, setSelectedId] = useState<string>(containers[0]?.id || '');
  const [logs, setLogs] = useState<string>('');
  const [search, setSearch] = useState('');
  const [tail, setTail] = useState('200');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!selectedId && containers.length > 0) {
      setSelectedId(containers[0].id);
    }
  }, [containers]);

  const loadLogs = async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const text = await api.getContainerLogs(selectedId, parseInt(tail, 10), true);
      setLogs(text);
    } catch (err: any) {
      setLogs(`Fehler beim Laden: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLogs();
  }, [selectedId, tail]);

  const filteredLogs = logs
    .split('\n')
    .filter((l) => !search || l.toLowerCase().includes(search.toLowerCase()))
    .join('\n');

  const selectedContainer = containers.find((c) => c.id === selectedId);

  const downloadLogs = () => {
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${selectedContainer?.name || 'container'}-logs.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-semibold text-white tracking-tight">Logs</h2>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Live-Ausgabe sämtlicher Container mit Suche und Export.
          </p>
        </div>

        {/* Container Selector Dropdown */}
        <div className="flex items-center gap-2 self-start sm:self-auto">
          <select
            value={selectedId}
            onChange={(e) => setSelectedId(e.target.value)}
            className="px-2.5 py-1.5 bg-[#181818] border border-[#282828] rounded-md text-xs font-mono text-neutral-200 focus:outline-none focus:border-neutral-400"
          >
            {containers.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.shortId})
              </option>
            ))}
          </select>

          <select
            value={tail}
            onChange={(e) => setTail(e.target.value)}
            className="px-2.5 py-1.5 bg-[#181818] border border-[#282828] rounded-md text-xs font-mono text-neutral-200 focus:outline-none focus:border-neutral-400"
          >
            <option value="100">100 Zeilen</option>
            <option value="250">250 Zeilen</option>
            <option value="500">500 Zeilen</option>
            <option value="1000">1000 Zeilen</option>
          </select>

          <button
            onClick={loadLogs}
            disabled={loading}
            className="p-1.5 rounded-md bg-[#181818] border border-[#282828] hover:bg-[#222222] text-neutral-400 hover:text-white transition-colors"
            title="Aktualisieren"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Terminal View */}
      <div className="rounded-lg bg-[#121212] border border-[#282828] overflow-hidden flex flex-col h-[560px]">
        <div className="p-3 bg-[#181818] border-b border-[#282828] flex items-center justify-between gap-3">
          <div className="relative flex-1 max-w-md">
            <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-neutral-500" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Logs filtern..."
              className="w-full pl-8 pr-3 py-1.5 bg-[#121212] border border-[#282828] rounded-md text-xs text-neutral-200 placeholder-neutral-500 focus:outline-none focus:border-neutral-400 font-mono"
            />
          </div>

          <button
            onClick={downloadLogs}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#181818] border border-[#282828] hover:bg-[#222222] text-xs text-neutral-300 transition-colors"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Exportieren</span>
          </button>
        </div>

        <div className="flex-1 p-4 font-mono text-xs text-neutral-300 overflow-y-auto whitespace-pre-wrap leading-relaxed select-text">
          {filteredLogs || 'Keine Logs für diesen Container verfügbar.'}
        </div>
      </div>
    </div>
  );
};

