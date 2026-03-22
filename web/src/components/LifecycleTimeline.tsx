import { Check, Circle, Loader2 } from 'lucide-react';
import { cn } from '../utils';

interface LifecycleTimelineProps {
  currentState: string;
}

const STATES = ['pending', 'running', 'stopped', 'terminated'];

export function LifecycleTimeline({ currentState }: LifecycleTimelineProps) {
  // Normalize state
  const state = currentState.toLowerCase();
  
  // Calculate current step index
  let currentIndex = STATES.indexOf(state);
  if (state === 'creating' || state === 'initializing') currentIndex = 0; // map to pending
  if (state === 'shutting-down' || state === 'stopping') currentIndex = 2; // map to stopped
  if (currentIndex === -1) currentIndex = 0; // default unknown to start

  return (
    <div className="relative pl-6 py-4">
      {/* Vertical line background */}
      <div className="absolute left-[11px] top-6 bottom-6 w-0.5 bg-border rounded-full" />
      
      {/* Animated progress line */}
      <div 
        className="absolute left-[11px] top-6 w-0.5 bg-primary rounded-full transition-all duration-1000 ease-in-out"
        style={{ height: `${(currentIndex / Math.max(1, STATES.length - 1)) * 100}%` }}
      />

      <div className="space-y-8 relative z-10">
        {STATES.map((step, index) => {
          const isCompleted = index < currentIndex;
          const isCurrent = index === currentIndex;
          

          let Icon = Circle;
          if (isCompleted) Icon = Check;
          if (isCurrent && state !== 'running' && state !== 'terminated' && state !== 'stopped') Icon = Loader2;

          return (
            <div key={step} className="flex gap-4 items-start group">
              <div className={cn(
                "relative flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2 transition-colors duration-500 bg-background",
                isCompleted ? "border-primary bg-primary text-primary-foreground" :
                isCurrent ? "border-primary text-primary shadow-[0_0_10px_has(var(--primary))]" :
                "border-border text-muted-foreground"
              )}>
                <Icon size={12} className={cn(isCurrent && Icon === Loader2 && "animate-spin")} />
              </div>
              
              <div className="flex flex-col">
                <span className={cn(
                  "text-sm font-semibold uppercase tracking-wider transition-colors duration-300",
                  (isCompleted || isCurrent) ? "text-foreground" : "text-muted-foreground"
                )}>
                  {step}
                </span>
                <span className="text-xs text-muted-foreground mt-1">
                  {isCurrent && state !== step ? `Transitioning: ${state}` : 
                   isCompleted ? 'Completed phase' : 
                   isCurrent ? 'Current phase' : 
                   'Pending phase'}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
