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
    <div className="space-y-4 animate-in fade-in duration-200">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-lg bg-[#151b26] border border-[#232d3f]">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-md bg-[#0e121b] border border-[#232d3f] text-blue-400">
            <Terminal className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-100 tracking-tight">Zentraler Log-Viewer</h3>
            <p className="text-xs text-slate-400 font-mono mt-0.5">
              Live-Ausgabe sämtlicher Container mit Suche und Export
            </p>
          </div>
        </div>

        {/* Container Selector Dropdown */}
        <div className="flex items-center gap-2">
          <select
            value={selectedId}
            onChange={(e) => setSelectedId(e.target.value)}
            className="px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs font-mono text-slate-200 focus:outline-none focus:border-blue-500"
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
            className="px-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs font-mono text-slate-200 focus:outline-none focus:border-blue-500"
          >
            <option value="100">100 Zeilen</option>
            <option value="250">250 Zeilen</option>
            <option value="500">500 Zeilen</option>
            <option value="1000">1000 Zeilen</option>
          </select>

          <button
            onClick={loadLogs}
            disabled={loading}
            className="p-1.5 rounded-md bg-[#0e121b] border border-[#232d3f] hover:bg-[#1c2433] text-slate-300 transition-colors"
            title="Aktualisieren"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Terminal View */}
      <div className="rounded-lg bg-[#0e121b] border border-[#232d3f] overflow-hidden flex flex-col h-[560px] shadow-xs">
        <div className="p-3 bg-[#151b26] border-b border-[#232d3f] flex items-center justify-between gap-3">
          <div className="relative flex-1 max-w-md">
            <Search className="w-4 h-4 absolute left-3 top-2 text-slate-500" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Logs filtern..."
              className="w-full pl-9 pr-3 py-1.5 bg-[#0e121b] border border-[#232d3f] rounded-md text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500 font-mono"
            />
          </div>

          <button
            onClick={downloadLogs}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#0e121b] border border-[#232d3f] hover:bg-[#1c2433] text-xs text-slate-300 transition-colors font-mono"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Exportieren</span>
          </button>
        </div>

        <div className="flex-1 p-4 font-mono text-xs text-slate-300 overflow-y-auto whitespace-pre-wrap leading-relaxed select-text">
          {filteredLogs || 'Keine Logs für diesen Container verfügbar.'}
        </div>
      </div>
    </div>
  );
};

