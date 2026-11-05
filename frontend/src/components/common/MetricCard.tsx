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
        'relative overflow-hidden rounded-lg bg-[#181818] border border-[#282828] p-4 transition-all duration-150 hover:border-[#383838]',
        alertThresholdExceeded && 'border-rose-800/80 bg-rose-950/20',
        className
      )}
    >
      <div className="flex items-center justify-between text-neutral-400 mb-2">
        <span className="text-xs font-medium text-neutral-400">{title}</span>
        {icon && <div className="text-neutral-500">{icon}</div>}
      </div>

      <div className="flex items-baseline gap-1 my-1">
        {isAvailable ? (
          <>
            <span className="text-2xl font-semibold tracking-tight text-white tabular-nums font-sans">
              {formattedValue ?? (gauge?.value !== null && gauge?.value !== undefined ? gauge.value : '—')}
            </span>
            {unit && <span className="text-xs text-neutral-400 font-normal ml-0.5">{unit}</span>}
          </>
        ) : isPending ? (
          <div className="flex items-center gap-1.5 text-xs text-neutral-400">
            <Clock className="w-3.5 h-3.5 animate-spin text-neutral-500" />
            <span>Erfasse Messwert...</span>
          </div>
        ) : isUnsupported ? (
          <div className="flex items-center gap-1.5 text-xs text-neutral-500">
            <AlertCircle className="w-3.5 h-3.5" />
            <span>Nicht unterstützt</span>
          </div>
        ) : (
          <div className="flex items-center gap-1.5 text-xs text-neutral-500" title={gauge?.reason}>
            <AlertCircle className="w-3.5 h-3.5" />
            <span>Nicht verfügbar</span>
          </div>
        )}
      </div>

      {progressPercent !== undefined && isAvailable && (
        <div className="w-full bg-[#262626] h-1.5 rounded-full overflow-hidden mt-2.5">
          <div
            className={cn(
              'h-full transition-all duration-300 rounded-full',
              progressPercent > 85 ? 'bg-rose-500' : progressPercent > 70 ? 'bg-amber-500' : 'bg-neutral-300'
            )}
            style={{ width: `${Math.min(100, Math.max(0, progressPercent))}%` }}
          />
        </div>
      )}

      {subtitle && <p className="text-xs text-neutral-500 mt-2 font-normal truncate">{subtitle}</p>}
    </div>
  );
};

