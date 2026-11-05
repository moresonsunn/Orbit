import React from 'react';
import { cn } from '../../lib/utils';

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className }) => {
  const s = status.toLowerCase();

  let bg = 'bg-slate-800 text-slate-300 border-slate-700';
  let dot = 'bg-slate-500';
  let label = status;

  if (s === 'running') {
    bg = 'bg-emerald-950/50 text-emerald-300 border-emerald-800/60';
    dot = 'bg-emerald-400 animate-pulse';
    label = 'Läuft';
  } else if (s === 'exited' || s === 'stopped') {
    bg = 'bg-slate-900 text-slate-400 border-slate-800';
    dot = 'bg-slate-500';
    label = 'Gestoppt';
  } else if (s === 'unhealthy') {
    bg = 'bg-rose-950/60 text-rose-300 border-rose-800/80';
    dot = 'bg-rose-500 animate-ping';
    label = 'Unhealthy';
  } else if (s === 'healthy') {
    bg = 'bg-emerald-950/40 text-emerald-400 border-emerald-800/40';
    dot = 'bg-emerald-400';
    label = 'Healthy';
  } else if (s === 'paused') {
    bg = 'bg-amber-950/50 text-amber-300 border-amber-800/60';
    dot = 'bg-amber-400';
    label = 'Pausiert';
  } else if (s === 'restarting') {
    bg = 'bg-indigo-950/50 text-indigo-300 border-indigo-800/60';
    dot = 'bg-indigo-400 animate-spin';
    label = 'Neustart...';
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border tracking-wide uppercase',
        bg,
        className
      )}
    >
      <span className={cn('w-1.5 h-1.5 rounded-full', dot)} />
      {label}
    </span>
  );
};

