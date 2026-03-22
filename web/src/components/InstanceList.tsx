import { useState, useMemo } from 'react';
import { useInstances } from '../hooks/useInstances';
import { InstanceCard } from './InstanceCard';
import { Search, Loader2, RefreshCw, Filter, DollarSign, Play, Square, Trash2 } from 'lucide-react';
import { cn } from '../utils';

interface InstanceListProps {
  onSelectInstance: (instanceId: string, namespace: string) => void;
  onRefresh: () => void;
  refreshing: boolean;
}

const COST_MAP: Record<string, number> = {
  't3.micro': 7.60,
  't3.small': 15.20,
  't3.medium': 30.40,
  'm5.large': 70.08,
};

export function InstanceList({ onSelectInstance, onRefresh, refreshing }: InstanceListProps) {
  const { instances, loading, error } = useInstances();
  const [searchTerm, setSearchTerm] = useState('');
  const [regionFilter, setRegionFilter] = useState('all');
  const [stateFilter, setStateFilter] = useState('all');
  const [selectedInstances, setSelectedInstances] = useState<Set<string>>(new Set());

  const estimatedMonthlyCost = useMemo(() => {
    return instances.filter(i => i.state === 'running' || i.state === 'pending').reduce((total, inst) => {
      return total + (COST_MAP[inst.instanceType] || 15.00); 
    }, 0);
  }, [instances]);

  const filteredInstances = useMemo(() => {
    return instances.filter(i => {
      const matchesSearch = i.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                            (i.instanceID && i.instanceID.toLowerCase().includes(searchTerm.toLowerCase()));
      const matchesRegion = regionFilter === 'all' || i.region === regionFilter;
      const matchesState = stateFilter === 'all' || i.state === stateFilter;
      return matchesSearch && matchesRegion && matchesState;
    });
  }, [instances, searchTerm, regionFilter, stateFilter]);

  const handleToggleSelect = (id: string, e: React.MouseEvent | React.ChangeEvent) => {
    e.stopPropagation();
    const newSet = new Set(selectedInstances);
    if (newSet.has(id)) newSet.delete(id);
    else newSet.add(id);
    setSelectedInstances(newSet);
  };

  if (loading) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center text-muted-foreground">
        <Loader2 className="w-8 h-8 animate-spin text-primary mb-4" />
        <p>Connecting to Operator...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center text-destructive">
        <p className="text-lg font-medium">Failed to load instances</p>
        <p className="text-sm opacity-80 mt-2">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex flex-col lg:flex-row justify-between items-start lg:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">Instances</h1>
          <p className="text-muted-foreground mt-1">Manage your operator-provisioned EC2 instances.</p>
        </div>
        
        <div className="flex items-center gap-4 bg-primary/10 text-primary px-4 py-2 rounded-lg border border-primary/20">
          <DollarSign size={20} />
          <div>
            <p className="text-xs uppercase font-bold tracking-wider opacity-80">Est. Monthly Cost</p>
            <p className="text-lg font-bold">${estimatedMonthlyCost.toFixed(2)}</p>
          </div>
        </div>
      </div>

      <div className="glass p-4 rounded-xl border border-border flex flex-col sm:flex-row flex-wrap gap-4 items-center justify-between">
        <div className="flex items-center gap-3 w-full sm:w-auto flex-wrap">
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input 
              type="text" 
              placeholder="Search instances..." 
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-background/50 border border-border rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all placeholder:text-muted-foreground/50"
            />
          </div>
          
          <div className="flex items-center gap-2 border border-border bg-background/50 rounded-lg px-3 py-2">
            <Filter size={14} className="text-muted-foreground" />
            <select 
              value={regionFilter} 
              onChange={e => setRegionFilter(e.target.value)}
              className="bg-transparent text-sm focus:outline-none text-foreground"
            >
              <option value="all">All Regions</option>
              <option value="us-east-1">us-east-1</option>
              <option value="eu-central-1">eu-central-1</option>
            </select>
          </div>

          <div className="flex items-center gap-2 border border-border bg-background/50 rounded-lg px-3 py-2">
            <select 
              value={stateFilter} 
              onChange={e => setStateFilter(e.target.value)}
              className="bg-transparent text-sm focus:outline-none text-foreground"
            >
              <option value="all">All States</option>
              <option value="running">running</option>
              <option value="stopped">stopped</option>
              <option value="pending">pending</option>
            </select>
          </div>
        </div>

        <div className="flex items-center gap-3 w-full sm:w-auto justify-end">
          {selectedInstances.size > 0 && (
            <div className="flex items-center gap-2 animate-in slide-in-from-right-4">
              <span className="text-sm font-medium mr-2">{selectedInstances.size} selected</span>
              <button className="p-2 bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20 rounded-lg transition-colors title='Start selected'">
                <Play size={16} />
              </button>
              <button className="p-2 bg-amber-500/10 text-amber-500 hover:bg-amber-500/20 rounded-lg transition-colors title='Stop selected'">
                <Square size={16} />
              </button>
              <button className="p-2 bg-destructive/10 text-destructive hover:bg-destructive/20 rounded-lg transition-colors title='Terminate selected'">
                <Trash2 size={16} />
              </button>
              <div className="w-px h-6 bg-border mx-1" />
            </div>
          )}
          <button 
            onClick={onRefresh}
            disabled={refreshing}
            className="p-2.5 bg-card hover:bg-card/80 border border-border rounded-lg text-muted-foreground hover:text-primary transition-all disabled:opacity-50"
            title="Refresh instances"
          >
            <RefreshCw size={18} className={cn(refreshing && "animate-spin")} />
          </button>
          <div className="relative w-full sm:w-72">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input 
              type="text" 
              placeholder="Search instances..." 
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-card/50 border border-border rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-all placeholder:text-muted-foreground/50"
            />
          </div>
        </div>
      </div>

      {filteredInstances.length === 0 ? (
        <div className="glass rounded-xl p-12 text-center text-muted-foreground border-dashed border-2">
          <p>No instances found matching your search.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6 auto-rows-fr">
          {filteredInstances.map((instance) => (
            <InstanceCard 
              key={`${instance.namespace}-${instance.name}`} 
              instance={instance} 
              onClick={() => onSelectInstance(instance.name, instance.namespace)}
              selected={selectedInstances.has(`${instance.namespace}/${instance.name}`)}
              onToggleSelect={(e) => handleToggleSelect(`${instance.namespace}/${instance.name}`, e)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
