import { cn } from '../utils';
import { Loader2 } from 'lucide-react';

interface StatusBadgeProps {
  state: string;
}

export function StatusBadge({ state }: StatusBadgeProps) {
  const s = state.toLowerCase();

  const getStatusStyles = () => {
    switch (s) {
      case 'running':
        return 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-[0_0_15px_-5px_rgba(16,185,129,0.3)]';
      case 'stopped':
        return 'bg-amber-500/10 text-amber-400 border border-amber-500/20';
      case 'terminated':
        return 'bg-rose-500/10 text-rose-400 border border-rose-500/20';
      case 'pending':
      case 'creating':
        return 'bg-sky-500/10 text-sky-400 border border-sky-500/20';
      case 'unknown':
      default:
        return 'bg-slate-500/10 text-slate-400 border border-slate-500/20';
    }
  };

  const getDotStyles = () => {
    switch (s) {
      case 'running': return 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.8)] animate-pulse-subtle';
      case 'stopped': return 'bg-amber-500';
      case 'terminated': return 'bg-rose-500';
      case 'pending':
      case 'creating': return 'bg-sky-500 animate-pulse';
      default: return 'bg-slate-500';
    }
  };

  return (
    <div className={cn("inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold uppercase tracking-wider", getStatusStyles())}>
      <span className={cn("w-2 h-2 rounded-full mr-2", getDotStyles())} />
      {s === 'pending' || s === 'creating' ? (
        <span className="flex items-center">
          <Loader2 className="w-3 h-3 mr-1.5 animate-spin" />
          {s}
        </span>
      ) : (
        s
      )}
    </div>
  );
}
