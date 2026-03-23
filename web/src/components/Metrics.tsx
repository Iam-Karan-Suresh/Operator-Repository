import { Activity, Cpu, Network, HardDrive } from 'lucide-react';
import { useState, useEffect } from 'react';

const API_URL = import.meta.env.VITE_API_URL || '';

interface Stats {
  reconciliationCount: number;
  instanceCount: number;
  apiLatency: number;
}

export function Metrics() {
  const [stats, setStats] = useState<Stats>({ reconciliationCount: 0, instanceCount: 0, apiLatency: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const resp = await fetch(`${API_URL}/api/stats`);
        if (resp.ok) {
          const data = await resp.json();
          setStats(data);
        }
      } catch (err) {
        console.error("Failed to fetch stats", err);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 10000); // refresh every 10s
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center">
            <Activity className="mr-3 text-primary" /> Metrics Overview
          </h1>
          <p className="text-muted-foreground mt-1">Aggregated statistics and resource monitoring.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mt-6">
        <MetricCard 
          icon={<Activity />} 
          title="Total Reconciliations" 
          value={loading ? "..." : stats.reconciliationCount.toLocaleString()} 
          change="+Live" 
        />
        <MetricCard 
          icon={<Cpu />} 
          title="Active Instances" 
          value={loading ? "..." : stats.instanceCount.toString()} 
          change="Real-time" 
        />
        <MetricCard 
          icon={<Network />} 
          title="API Latency (avg)" 
          value={loading ? "..." : `${stats.apiLatency.toFixed(1)}ms`} 
          change={stats.apiLatency > 50 ? "High" : "Optimal"} 
        />
        <MetricCard icon={<HardDrive />} title="Total Storage" value="500 GB" change="+50 GB" />
      </div>

      <div className="mt-8 glass rounded-xl p-8 border border-border flex flex-col items-center justify-center text-center min-h-[300px]">
        <Activity size={48} className="text-muted-foreground mb-4 opacity-50" />
        <h3 className="text-xl font-semibold mb-2">Detailed Observability</h3>
        <p className="text-muted-foreground max-w-md">
          The production observability stack (Prometheus + Grafana + OpenTelemetry) is now integrated. 
          Use the Helm-deployed Grafana dashboard for full-fidelity tracing and long-term metric analysis.
        </p>
      </div>
    </div>
  );
}

function MetricCard({ icon, title, value, change }: { icon: React.ReactNode, title: string, value: string, change: string }) {
  const isPositive = change.startsWith('+');
  const isNeutral = change === '0%';
  return (
    <div className="glass rounded-xl p-6 border border-border hover:border-primary/30 transition-colors">
      <div className="flex items-center space-x-3 mb-4 text-muted-foreground">
        <span className="text-primary">{icon}</span>
        <h3 className="font-medium text-sm">{title}</h3>
      </div>
      <div className="flex items-end justify-between">
        <span className="text-3xl font-bold text-foreground">{value}</span>
        <span className={`text-sm font-medium ${isNeutral ? 'text-muted-foreground' : isPositive ? 'text-emerald-500' : 'text-blue-500'}`}>
          {change} 
        </span>
      </div>
    </div>
  );
}
