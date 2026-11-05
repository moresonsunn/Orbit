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

  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-lg bg-[#151b26] border border-[#232d3f] p-4 transition-all duration-150 hover:border-[#334155] shadow-xs',
        alertThresholdExceeded && 'border-rose-500/50 bg-rose-950/20',
        className
      )}
    >
      <div className="flex items-center justify-between text-slate-400 mb-2">
        <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{title}</span>
        {icon && <div className="p-1 rounded bg-[#0e121b] border border-[#232d3f] text-slate-400">{icon}</div>}
      </div>

      <div className="flex items-baseline gap-1 my-1">
        {isAvailable ? (
          <>
            <span className="text-2xl font-bold tracking-tight text-white tabular-nums">
              {formattedValue ?? (gauge?.value !== null && gauge?.value !== undefined ? gauge.value : '—')}
            </span>
            {unit && <span className="text-xs text-slate-400 font-medium ml-0.5">{unit}</span>}
          </>
        ) : isPending ? (
          <div className="flex items-center gap-1.5 text-xs text-amber-400 bg-amber-500/10 px-2 py-1 rounded border border-amber-500/20">
            <Clock className="w-3.5 h-3.5 animate-spin" />
            <span>Erfasse Messwert...</span>
          </div>
        ) : isUnsupported ? (
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-[#0e121b] px-2 py-1 rounded border border-[#232d3f]">
            <AlertCircle className="w-3.5 h-3.5 text-slate-500" />
            <span>Nicht unterstützt</span>
          </div>
        ) : (
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-[#0e121b] px-2 py-1 rounded border border-[#232d3f]" title={gauge?.reason}>
            <AlertCircle className="w-3.5 h-3.5 text-slate-500" />
            <span>Nicht verfügbar</span>
          </div>
        )}
      </div>

      {progressPercent !== undefined && isAvailable && (
        <div className="w-full bg-[#0e121b] border border-[#232d3f]/60 h-2 rounded-full overflow-hidden mt-2.5">
          <div
            className={cn(
              'h-full transition-all duration-300 rounded-full',
              progressPercent > 85 ? 'bg-rose-500' : progressPercent > 70 ? 'bg-amber-500' : 'bg-blue-500'
            )}
            style={{ width: `${Math.min(100, Math.max(0, progressPercent))}%` }}
          />
        </div>
      )}

      {subtitle && <p className="text-xs text-slate-400 mt-2 font-normal truncate">{subtitle}</p>}
    </div>
  );
};

