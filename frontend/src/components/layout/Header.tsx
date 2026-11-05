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
    <header className="h-14 border-b border-[#222222] bg-[#141414] px-6 flex items-center justify-between z-20 font-sans">
      <div>
        <h1 className="text-sm font-semibold tracking-tight text-white flex items-center gap-2">
          {title}
        </h1>
        {subtitle && <p className="text-xs text-neutral-400 font-normal mt-0.5">{subtitle}</p>}
      </div>

      <div className="flex items-center gap-2.5">
        {/* Container count indicator */}
        <div className="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-[#1a1a1a] border border-[#262626] text-xs font-normal text-neutral-300">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
          <span>
            {runningContainersCount} von {totalContainersCount} aktiv
          </span>
        </div>

        {/* Realtime WebSocket status */}
        <div
          className="flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-normal text-neutral-400"
          title={wsConnected ? 'Echtzeit-Verbindung aktiv' : 'Echtzeit-Verbindung getrennt'}
        >
          <span className={cn('w-1.5 h-1.5 rounded-full', wsConnected ? 'bg-emerald-500' : 'bg-neutral-600')} />
          <span>{wsConnected ? 'Live' : 'Getrennt'}</span>
        </div>

        {/* Alerts Button */}
        <button
          onClick={onOpenAlerts}
          className="relative p-1.5 rounded-md text-neutral-400 hover:text-white hover:bg-white/[0.05] transition-colors"
          title="Benachrichtigungen und Warnungen"
        >
          <Bell className="w-4 h-4" />
          {unreadAlerts.length > 0 && (
            <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-3.5 h-3.5 px-1 rounded-full bg-rose-500 text-white text-[9px] font-bold">
              {unreadAlerts.length}
            </span>
          )}
        </button>
      </div>
    </header>
  );
};

