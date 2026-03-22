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
        return 'bg-success/10 text-success border border-success/20';
      case 'stopped':
        return 'bg-warning/10 text-warning border border-warning/20';
      case 'terminated':
        return 'bg-destructive/10 text-destructive border border-destructive/20';
      case 'pending':
      case 'creating':
        return 'bg-primary/10 text-primary border border-primary/20';
      case 'unknown':
      default:
        return 'bg-muted text-muted-foreground border border-border';
    }
  };

  const getDotStyles = () => {
    switch (s) {
      case 'running': return 'bg-success shadow-[0_0_8px_has(var(--success))]';
      case 'stopped': return 'bg-warning';
      case 'terminated': return 'bg-destructive';
      case 'pending':
      case 'creating': return 'bg-primary animate-pulse';
      default: return 'bg-muted-foreground';
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
