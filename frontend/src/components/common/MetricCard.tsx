import React from 'react';
import { Gauge } from '../../types';
import { cn } from '../../lib/utils';
import { AlertCircle, Clock } from 'lucide-react';

interface MetricCardProps {
  title: string;
  gauge?: Gauge;
  formattedValue?: string;
  unit?: string;
  subtitle?: string;
  icon?: React.ReactNode;
  progressPercent?: number;
  className?: string;
  alertThresholdExceeded?: boolean;
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  gauge,
  formattedValue,
  unit,
  subtitle,
  icon,
  progressPercent,
  className,
  alertThresholdExceeded,
}) => {
  const isAvailable = !gauge || gauge.status === 'ok';
  const isPending = gauge?.status === 'pending';
  const isUnsupported = gauge?.status === 'unsupported';
  const isUnavailable = gauge?.status === 'unavailable' || gauge?.status === 'no_permission';

  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-xl bg-slate-900/60 border border-slate-800/80 p-4 transition-all duration-200 hover:border-slate-700/80 shadow-sm',
        alertThresholdExceeded && 'border-rose-500/50 bg-rose-950/10',
        className
      )}
    >
      <div className="flex items-center justify-between text-slate-400 mb-2">
        <span className="text-xs font-medium uppercase tracking-wider text-slate-400">{title}</span>
        {icon && <div className="text-slate-400">{icon}</div>}
      </div>

      <div className="flex items-baseline gap-1.5 my-1">
        {isAvailable ? (
          <>
            <span className="text-2xl font-bold font-mono tracking-tight text-white">
              {formattedValue ?? (gauge?.value !== null && gauge?.value !== undefined ? gauge.value : '—')}
            </span>
            {unit && <span className="text-xs text-slate-400 font-medium">{unit}</span>}
          </>
        ) : isPending ? (
          <div className="flex items-center gap-1.5 text-xs text-amber-400 bg-amber-950/40 px-2 py-1 rounded border border-amber-800/50">
            <Clock className="w-3.5 h-3.5 animate-spin" />
            <span>Erfasse Messwert...</span>
          </div>
        ) : isUnsupported ? (
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-800/60 px-2 py-1 rounded border border-slate-700/50">
            <AlertCircle className="w-3.5 h-3.5 text-slate-400" />
            <span>Nicht unterstützt</span>
          </div>
        ) : (
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-800/40 px-2 py-1 rounded border border-slate-700/40" title={gauge?.reason}>
            <AlertCircle className="w-3.5 h-3.5 text-slate-400" />
            <span>Nicht verfügbar</span>
          </div>
        )}
      </div>

      {progressPercent !== undefined && isAvailable && (
        <div className="w-full bg-slate-800/80 h-1.5 rounded-full overflow-hidden mt-2.5">
          <div
            className={cn(
              'h-full transition-all duration-500 rounded-full',
              progressPercent > 85 ? 'bg-rose-500' : progressPercent > 70 ? 'bg-amber-500' : 'bg-cyan-500'
            )}
            style={{ width: `${Math.min(100, Math.max(0, progressPercent))}%` }}
          />
        </div>
      )}

      {subtitle && <p className="text-xs text-slate-400 mt-2 font-mono truncate">{subtitle}</p>}
    </div>
  );
};

