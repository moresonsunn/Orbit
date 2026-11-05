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
    <header className="h-16 border-b border-slate-800/70 bg-[#0b0f17]/85 backdrop-blur-md px-6 flex items-center justify-between z-20">
      <div>
        <h1 className="text-base font-semibold tracking-tight text-white flex items-center gap-2">
          {title}
        </h1>
        {subtitle && <p className="text-xs text-slate-400 font-normal mt-0.5">{subtitle}</p>}
      </div>

      <div className="flex items-center gap-3">
        {/* Container count indicator */}
        <div className="hidden sm:flex items-center gap-2 px-3 py-1 rounded-full bg-slate-900/80 border border-slate-800 text-xs font-medium text-slate-300">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
          <span>
            {runningContainersCount} / {totalContainersCount} Aktiv
          </span>
        </div>

        {/* Realtime WebSocket status */}
        <div
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border',
            wsConnected
              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
              : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
          )}
          title={wsConnected ? 'Echtzeit-Verbindung aktiv' : 'Echtzeit-Verbindung getrennt'}
        >
          <span className={cn('w-1.5 h-1.5 rounded-full', wsConnected ? 'bg-emerald-400' : 'bg-rose-400')} />
          <span>{wsConnected ? 'LIVE' : 'OFFLINE'}</span>
        </div>

        {/* Alerts Button */}
        <button
          onClick={onOpenAlerts}
          className="relative p-2 rounded-lg bg-slate-900/80 border border-slate-800 text-slate-300 hover:text-white hover:bg-slate-800/80 transition-colors"
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

