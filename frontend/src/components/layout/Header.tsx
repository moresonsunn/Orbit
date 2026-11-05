import React from 'react';
import { Bell, Radio } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Alert } from '../../types';

interface HeaderProps {
  title: string;
  subtitle?: string;
  wsConnected: boolean;
  alerts: Alert[];
  onOpenAlerts?: () => void;
  runningContainersCount?: number;
  totalContainersCount?: number;
}

export const Header: React.FC<HeaderProps> = ({
  title,
  subtitle,
  wsConnected,
  alerts,
  onOpenAlerts,
  runningContainersCount = 0,
  totalContainersCount = 0,
}) => {
  const unreadAlerts = alerts.filter((a) => !a.acknowledged && !a.resolvedAt);

  return (
    <header className="h-16 border-b border-slate-800/80 bg-[#070b14]/80 backdrop-blur-md px-6 flex items-center justify-between z-20">
      <div>
        <h1 className="text-lg font-bold tracking-tight text-white flex items-center gap-2">
          {title}
        </h1>
        {subtitle && <p className="text-xs text-slate-400 font-mono -mt-0.5">{subtitle}</p>}
      </div>

      <div className="flex items-center gap-4">
        {/* Container count indicator */}
        <div className="hidden sm:flex items-center gap-2 px-3 py-1 rounded-full bg-slate-900 border border-slate-800 text-xs font-mono text-slate-300">
          <span className="w-2 h-2 rounded-full bg-emerald-400" />
          <span>
            {runningContainersCount} / {totalContainersCount} Aktiv
          </span>
        </div>

        {/* Realtime WebSocket status */}
        <div
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-mono border',
            wsConnected
              ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/50'
              : 'bg-rose-950/40 text-rose-400 border-rose-800/50'
          )}
          title={wsConnected ? 'Echtzeit-Verbindung aktiv' : 'Echtzeit-Verbindung unterbrochen'}
        >
          <Radio className={cn('w-3.5 h-3.5', wsConnected && 'animate-pulse')} />
          <span className="hidden md:inline">{wsConnected ? 'LIVE' : 'OFFLINE'}</span>
        </div>

        {/* Alerts Button */}
        <button
          onClick={onOpenAlerts}
          className="relative p-2 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-white hover:bg-slate-800/80 transition-colors"
          title="Benachrichtigungen und Warnungen"
        >
          <Bell className="w-4 h-4" />
          {unreadAlerts.length > 0 && (
            <span className="absolute -top-1 -right-1 flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-rose-500 text-white text-[10px] font-bold">
              {unreadAlerts.length}
            </span>
          )}
        </button>
      </div>
    </header>
  );
};
