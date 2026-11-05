import React from 'react';
import { cn } from '../../lib/utils';

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className }) => {
  const s = status.toLowerCase();

  let dot = 'bg-neutral-500';
  let textColor = 'text-neutral-400';
  let label = status;

  if (s === 'running') {
    dot = 'bg-emerald-500';
    textColor = 'text-neutral-200';
    label = 'Läuft';
  } else if (s === 'exited' || s === 'stopped') {
    dot = 'bg-neutral-500';
    textColor = 'text-neutral-400';
    label = 'Gestoppt';
  } else if (s === 'unhealthy') {
    dot = 'bg-rose-500';
    textColor = 'text-rose-400';
    label = 'Unhealthy';
  } else if (s === 'healthy') {
    dot = 'bg-emerald-500';
    textColor = 'text-neutral-200';
    label = 'Healthy';
  } else if (s === 'paused') {
    dot = 'bg-amber-500';
    textColor = 'text-amber-400';
    label = 'Pausiert';
  } else if (s === 'restarting') {
    dot = 'bg-blue-500 animate-pulse';
    textColor = 'text-blue-400';
    label = 'Neustart...';
  }

  return (
    <span className={cn('inline-flex items-center gap-1.5 text-xs font-normal whitespace-nowrap', textColor, className)}>
      <span className={cn('w-1.5 h-1.5 rounded-full shrink-0', dot)} />
      <span>{label}</span>
    </span>
  );
};

