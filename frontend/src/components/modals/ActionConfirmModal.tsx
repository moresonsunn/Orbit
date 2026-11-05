import React from 'react';
import { AlertTriangle, X } from 'lucide-react';
import { cn } from '../../lib/utils';

interface ActionConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  affectedItems?: string[];
  confirmText?: string;
  isDestructive?: boolean;
  isLoading?: boolean;
}

export const ActionConfirmModal: React.FC<ActionConfirmModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  affectedItems,
  confirmText = 'Bestätigen',
  isDestructive = false,
  isLoading = false,
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="relative w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden p-6">
        <button
          onClick={onClose}
          disabled={isLoading}
          className="absolute top-4 right-4 p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
        >
          <X className="w-4 h-4" />
        </button>

        <div className="flex items-start gap-3">
          <div
            className={cn(
              'p-2.5 rounded-xl border shrink-0',
              isDestructive
                ? 'bg-rose-950/50 border-rose-800 text-rose-400'
                : 'bg-amber-950/50 border-amber-800 text-amber-400'
            )}
          >
            <AlertTriangle className="w-5 h-5" />
          </div>

          <div className="flex-1 pr-4">
            <h3 className="text-base font-bold text-white">{title}</h3>
            <p className="text-xs text-slate-400 mt-1 leading-relaxed">{description}</p>
          </div>
        </div>

        {affectedItems && affectedItems.length > 0 && (
          <div className="mt-4 p-3 bg-slate-950/70 border border-slate-800/80 rounded-xl">
            <span className="text-xs font-medium text-slate-400 block mb-1">
              Betroffene Elemente ({affectedItems.length}):
            </span>
            <div className="max-h-28 overflow-y-auto space-y-1 font-mono text-xs text-slate-300">
              {affectedItems.map((item, i) => (
                <div key={i} className="truncate bg-slate-900 px-2 py-0.5 rounded border border-slate-800/50">
                  {item}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="flex items-center justify-end gap-3 mt-6">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="px-4 py-2 rounded-lg text-xs font-medium text-slate-300 hover:bg-slate-800 transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={onConfirm}
            disabled={isLoading}
            className={cn(
              'px-4 py-2 rounded-lg text-xs font-semibold text-white transition-all shadow-md',
              isDestructive
                ? 'bg-rose-600 hover:bg-rose-500 shadow-rose-950/50'
                : 'bg-cyan-600 hover:bg-cyan-500 shadow-cyan-950/50'
            )}
          >
            {isLoading ? 'Ausführen...' : confirmText}
          </button>
        </div>
      </div>
    </div>
  );
};
