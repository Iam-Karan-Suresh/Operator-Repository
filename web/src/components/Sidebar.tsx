import { LayoutDashboard, Settings, Server, Activity } from 'lucide-react';
import { cn } from '../utils';

interface SidebarProps {
  className?: string;
  currentView: string;
  onSelectView: (view: string) => void;
  uiSettings?: {
    name: string;
    profession: string;
    team: string;
  };
}

export function Sidebar({ className, currentView, onSelectView, uiSettings }: SidebarProps) {
  const initials = uiSettings?.name
    ? uiSettings.name.split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase()
    : 'K';

  return (
    <aside className={cn("w-64 border-r border-border bg-card/50 backdrop-blur-md flex flex-col", className)}>
      <div className="h-16 flex items-center px-6 border-b border-border">
        <Server className="w-6 h-6 text-primary mr-3" />
        <span className="font-bold text-lg tracking-tight">EC2 Operator</span>
      </div>
      
      <nav className="flex-1 py-6 px-4 space-y-2">
        <NavItem icon={<LayoutDashboard size={20} />} label="Instances" active={currentView === 'instances'} onClick={() => onSelectView('instances')} />
        <NavItem icon={<Activity size={20} />} label="Metrics" active={currentView === 'metrics'} onClick={() => onSelectView('metrics')} />
        <NavItem icon={<Settings size={20} />} label="Settings" active={currentView === 'settings'} onClick={() => onSelectView('settings')} />
      </nav>

      <div className="p-4 border-t border-border mt-auto group cursor-pointer hover:bg-white/5 transition-colors">
        <div className="flex items-center space-x-3 px-2">
          <div className="relative">
            <div className="absolute inset-0 bg-primary/20 rounded-full animate-ping opacity-0 group-hover:opacity-100 transition-opacity" />
            <div className="w-8 h-8 rounded-full bg-primary/20 border border-primary/20 flex items-center justify-center text-primary font-semibold text-xs relative z-10 group-hover:ring-2 group-hover:ring-primary/50 group-hover:scale-110 transition-all duration-300">
              {initials}
            </div>
          </div>
          <div className="text-sm overflow-hidden">
            <p className="font-medium leading-none mb-1 truncate text-foreground/90 group-hover:text-foreground transition-colors">{uiSettings?.name || 'User Name'}</p>
            <p className="text-muted-foreground text-[10px] truncate group-hover:text-muted-foreground/80">{uiSettings?.team || 'Cloud Team'}</p>
          </div>
        </div>
      </div>
    </aside>
  );
}

function NavItem({ icon, label, active, onClick }: { icon: React.ReactNode; label: string; active?: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "w-full flex items-center px-3 py-2.5 rounded-md transition-all duration-200 group text-left",
        active 
          ? "bg-primary/10 text-primary font-medium" 
          : "text-muted-foreground hover:bg-white/5 hover:text-foreground"
      )}
    >
      <span className={cn(
        "mr-3 transition-colors", 
        active ? "text-primary" : "text-muted-foreground group-hover:text-foreground"
      )}>
        {icon}
      </span>
      {label}
    </button>
  );
}
