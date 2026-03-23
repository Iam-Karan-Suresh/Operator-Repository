import React from 'react';
import type { InstanceResponse } from '../types/instance';
import { StatusBadge } from './StatusBadge';
import { Server, Cpu, HardDrive, Globe, Clock, ArrowRight } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

interface InstanceCardProps {
  instance: InstanceResponse;
  onClick: () => void;
  selected: boolean;
  onToggleSelect: (e: React.MouseEvent | React.ChangeEvent) => void;
}

export function InstanceCard({ instance, onClick, selected, onToggleSelect }: InstanceCardProps) {
  return (
    <div 
      onClick={onClick}
      className="glass rounded-xl p-5 hover:bg-card/80 transition-all cursor-pointer group hover:scale-[1.02] border-border/50 hover:border-primary/30 relative overflow-hidden"
    >
      {/* Decorative gradient orb */}
      <div className="absolute -top-10 -right-10 w-32 h-32 bg-primary/10 rounded-full blur-3xl group-hover:bg-primary/20 transition-all" />

      {/* Selection Checkbox */}
      <div className="absolute top-4 right-4 z-20" onClick={e => e.stopPropagation()}>
          <input 
            type="checkbox" 
            checked={selected}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              e.stopPropagation();
              onToggleSelect(e);
            }}
            className="w-4 h-4 rounded border-border text-primary focus:ring-primary/50 transition-all cursor-pointer" 
          />
      </div>

      <div className="flex justify-between items-start mb-4 relative z-10 pr-8">
        <div className="flex items-center space-x-3">
          <div className="p-2.5 bg-primary/10 rounded-lg text-primary">
            <Server size={20} />
          </div>
          <div>
            <h3 className="font-semibold text-foreground text-base group-hover:text-primary transition-colors truncate w-full max-w-[12rem]">
              {instance.name}
            </h3>
            <p className="text-xs text-muted-foreground font-mono mt-0.5">{instance.instanceID || 'Pending ID'}</p>
          </div>
        </div>
        <StatusBadge state={instance.state || 'Unknown'} />
      </div>

      <div className="grid grid-cols-2 gap-y-3 gap-x-4 mt-6 text-sm relative z-10">
        <div className="flex items-center text-muted-foreground">
          <Cpu size={14} className="mr-2 opacity-70" />
          <span className="truncate">{instance.instanceType}</span>
        </div>
        <div className="flex items-center text-muted-foreground">
          <Globe size={14} className="mr-2 opacity-70" />
          <span className="truncate">{instance.publicIP || 'No Public IP'}</span>
        </div>
        <div className="flex items-center text-muted-foreground">
          <HardDrive size={14} className="mr-2 opacity-70" />
          <span className="truncate">{instance.region}</span>
        </div>
        <div className="flex items-center text-muted-foreground">
          <Clock size={14} className="mr-2 opacity-70" />
          <span className="truncate">{formatDistanceToNow(new Date(instance.createdAt))} ago</span>
        </div>
      </div>

      <div className="mt-5 pt-4 border-t border-border/50 flex justify-between items-center text-xs font-medium text-muted-foreground transition-colors relative z-10">
        <span>Namespace: {instance.namespace}</span>
        <div className="flex items-center space-x-3">
          <button 
            onClick={(e) => { e.stopPropagation(); onClick(); }}
            className="flex items-center hover:text-primary transition-colors py-1 px-2 rounded-md hover:bg-primary/10"
          >
            <ArrowRight size={14} className="mr-1" /> Details
          </button>
        </div>
      </div>
    </div>
  );
}
