import React from 'react';
import { cn } from '../../lib/utils';

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className }) => {
  const s = status.toLowerCase();

  let bg = 'bg-slate-800/70 text-slate-400 border-slate-700/50';
  let dot = 'bg-slate-500';
  let label = status;

  if (s === 'running') {
    bg = 'bg-emerald-500/10 text-emerald-400 border-emerald-500/25';
    dot = 'bg-emerald-400';
    label = 'Läuft';
  } else if (s === 'exited' || s === 'stopped') {
    bg = 'bg-slate-800/80 text-slate-400 border-slate-700/50';
    dot = 'bg-slate-500';
    label = 'Gestoppt';
  } else if (s === 'unhealthy') {
    bg = 'bg-rose-500/10 text-rose-400 border-rose-500/25';
    dot = 'bg-rose-400';
    label = 'Problem';
  } else if (s === 'healthy') {
    bg = 'bg-emerald-500/10 text-emerald-400 border-emerald-500/25';
    dot = 'bg-emerald-400';
    label = 'Healthy';
  } else if (s === 'paused') {
    bg = 'bg-amber-500/10 text-amber-400 border-amber-500/25';
    dot = 'bg-amber-400';
    label = 'Pausiert';
  } else if (s === 'restarting') {
    bg = 'bg-blue-500/10 text-blue-400 border-blue-500/25';
    dot = 'bg-blue-400 animate-pulse';
    label = 'Neustart...';
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border tracking-normal',
        bg,
        className
      )}
    >
      <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', dot)} />
      <span>{label}</span>
    </span>
  );
};

