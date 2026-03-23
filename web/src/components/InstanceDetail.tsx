import React, { useState, useEffect, useCallback } from 'react';
import type { InstanceResponse, EventResponse, LogResponse } from '../types/instance';
import { StatusBadge } from './StatusBadge';
import { LifecycleTimeline } from './LifecycleTimeline';
import { 
  ArrowLeft, 
  Server, 
  Cpu, 
  Terminal, 
  Key, 
  Shield, 
  Network, 
  RefreshCw, 
  MoreVertical, 
  FileJson, 
  Copy, 
  TerminalSquare, 
  X, 
  AlertCircle 
} from 'lucide-react';
import { format } from 'date-fns';
import { cn } from '../utils';
import { fetchInstanceEvents, fetchInstanceLogs } from '../api/client';

interface InstanceDetailProps {
  instance: InstanceResponse;
  onBack: () => void;
  onRefresh: () => void;
  refreshing: boolean;
}

export function InstanceDetail({ instance, onBack, onRefresh, refreshing }: InstanceDetailProps) {
  const [showLogs, setShowLogs] = useState(false);
  const [events, setEvents] = useState<EventResponse[]>([]);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [activeTab, setActiveTab] = useState<'events' | 'logs'>('logs');
  const [logs, setLogs] = useState<LogResponse[]>([]);
  const [loadingLogs, setLoadingLogs] = useState(false);

  const loadEvents = useCallback(async () => {
    setLoadingEvents(true);
    try {
      const data = await fetchInstanceEvents(instance.name, instance.namespace);
      setEvents(data);
    } catch (err) {
      console.error('Failed to fetch events:', err);
    } finally {
      setLoadingEvents(false);
    }
  }, [instance.name, instance.namespace]);

  const loadLogs = useCallback(async () => {
    setLoadingLogs(true);
    try {
      const data = await fetchInstanceLogs(instance.name, instance.namespace);
      setLogs(data);
    } catch (err) {
      console.error('Failed to fetch logs:', err);
    } finally {
      setLoadingLogs(false);
    }
  }, [instance.name, instance.namespace]);

  useEffect(() => {
    if (showLogs) {
      if (activeTab === 'events') {
        loadEvents();
      } else {
        loadLogs();
      }
    }
  }, [showLogs, activeTab, loadEvents, loadLogs]);

  return (
    <div className="space-y-6">
      <div className="bg-card/30 border border-border p-6 rounded-2xl mb-8 shadow-xl backdrop-blur-md">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center space-x-4">
            <button 
              onClick={onBack}
              className="p-3 bg-background hover:bg-muted border border-border rounded-xl text-muted-foreground hover:text-foreground transition-all flex items-center justify-center group shadow-sm"
              title="Back to list"
            >
              <ArrowLeft className="w-5 h-5 group-hover:-translate-x-1 transition-transform" />
            </button>
            
            <div>
              <div className="flex items-center space-x-3 mb-1">
                <h1 className="text-2xl font-bold text-foreground">
                  {instance.name}
                </h1>
                <StatusBadge state={instance.state} />
              </div>
              <p className="text-xs text-muted-foreground font-mono">
                {instance.namespace} <span className="mx-1 opacity-40">|</span> {instance.instanceID || 'Provisioning...'}
              </p>
            </div>
          </div>

          <div className="flex items-center space-x-3">
            <button 
              onClick={onRefresh}
              disabled={refreshing}
              className="p-3 bg-background hover:bg-muted border border-border rounded-xl text-muted-foreground hover:text-primary transition-all disabled:opacity-50 shadow-sm"
              title="Refresh logic"
            >
              <RefreshCw size={20} className={cn(refreshing && "animate-spin")} />
            </button>
            
            <button 
              id="view-logs-button"
              onClick={() => setShowLogs(!showLogs)}
              className={cn(
                "px-5 py-3 rounded-xl text-sm font-bold transition-all flex items-center space-x-2 shadow-lg border-2",
                showLogs 
                  ? "bg-primary text-primary-foreground border-primary" 
                  : "bg-primary/10 text-primary border-primary/20 hover:bg-primary/20"
              )}
            >
              <Terminal size={18} />
              <span>{showLogs ? 'Hide Instance Logs' : 'View Activity Logs'}</span>
            </button>

            <div className="relative group/actions">
              <button className="p-3 bg-background hover:bg-muted border border-border rounded-xl text-muted-foreground hover:text-foreground transition-all flex items-center justify-center shadow-sm">
                <MoreVertical className="w-5 h-5" />
              </button>
              <div className="absolute right-0 top-full mt-2 w-56 bg-card border border-border rounded-xl shadow-2xl opacity-0 invisible group-hover/actions:opacity-100 group-hover/actions:visible transition-all duration-200 z-50 overflow-hidden backdrop-blur-xl">
                <button 
                  onClick={() => navigator.clipboard.writeText(JSON.stringify(instance, null, 2))}
                  className="w-full flex items-center px-4 py-3 text-sm hover:bg-primary/10 text-muted-foreground hover:text-foreground transition-colors text-left"
                >
                  <FileJson className="w-4 h-4 mr-3 text-primary" /> Copy Raw JSON
                </button>
                <button 
                  onClick={() => navigator.clipboard.writeText(instance.instanceID || '')}
                  className="w-full flex items-center px-4 py-3 text-sm hover:bg-primary/10 text-muted-foreground hover:text-foreground transition-colors text-left border-t border-border/50"
                >
                  <Copy className="w-4 h-4 mr-3 text-primary" /> Copy Instance ID
                </button>
                <button 
                  onClick={() => alert(`Connect via SSH using:\nssh -i "key.pem" ec2-user@${instance.publicDNS || instance.publicIP || '[pending]'}`)}
                  className="w-full flex items-center px-4 py-3 text-sm hover:bg-primary/10 text-primary transition-colors text-left border-t border-border/50"
                >
                  <TerminalSquare className="w-4 h-4 mr-3" /> Connect (SSH)
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content Column */}
        <div className="lg:col-span-2 space-y-6">
          {/* Specifications Card */}
          <div className="glass rounded-xl p-6 border-border/50">
            <h2 className="text-lg font-semibold border-b border-border/50 pb-4 mb-5 flex items-center">
              <Cpu className="w-5 h-5 mr-2 text-primary" />
              Compute Specifications
            </h2>
            <div className="grid grid-cols-2 gap-6">
              <DetailItem label="Instance Type" value={instance.instanceType} />
              <DetailItem label="AMI ID" value={instance.amiId} copyable />
              <DetailItem label="Region" value={instance.region} />
              <DetailItem label="Availability Zone" value={instance.availabilityZone || 'Auto'} />
            </div>
          </div>

          {/* Networking Card */}
          <div className="glass rounded-xl p-6 border-border/50">
            <h2 className="text-lg font-semibold border-b border-border/50 pb-4 mb-5 flex items-center">
              <Network className="w-5 h-5 mr-2 text-primary" />
              Networking
            </h2>
            <div className="grid grid-cols-2 gap-6">
              <DetailItem label="Public IP" value={instance.publicIP || 'None'} copyable={!!instance.publicIP} />
              <DetailItem label="Private IP" value={instance.privateIP || 'Pending'} copyable={!!instance.privateIP} />
              <DetailItem label="Public DNS" value={instance.publicDNS || 'None'} copyable={!!instance.publicDNS} />
              <DetailItem label="Private DNS" value={instance.privateDNS || 'Pending'} copyable={!!instance.privateDNS} />
            </div>
          </div>

          {/* Configuration Card */}
          <div className="glass rounded-xl p-6 border-border/50">
            <h2 className="text-lg font-semibold border-b border-border/50 pb-4 mb-5 flex items-center">
              <Shield className="w-5 h-5 mr-2 text-primary" />
              Configuration
            </h2>
            <div className="flex flex-col space-y-4 text-sm">
              <div className="grid grid-cols-3 gap-4 border-b border-border/30 pb-3">
                <span className="text-muted-foreground font-medium flex items-center">
                  <Key className="w-4 h-4 mr-2" /> Tags
                </span>
                <div className="col-span-2 flex flex-wrap gap-2">
                  {instance.tags && Object.keys(instance.tags).length > 0 ? (
                    Object.entries(instance.tags).map(([k, v]) => (
                      <span key={k} className="px-2 py-1 bg-primary/10 text-primary hover:bg-primary/20 transition-colors border border-primary/20 rounded text-xs font-mono">
                        {k}: {v}
                      </span>
                    ))
                  ) : (
                    <span className="text-muted-foreground italic">No tags configured</span>
                  )}
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4 pt-1">
                <span className="text-muted-foreground font-medium flex items-center">
                  <Terminal className="w-4 h-4 mr-2" /> Creation Time
                </span>
                <div className="col-span-2 text-foreground font-medium">
                  {format(new Date(instance.createdAt), 'PPpp')} <span className="text-muted-foreground font-normal ml-2">({instance.age} ago)</span>
                </div>
              </div>
            </div>
          </div>

          {/* Logs / Events Panel */}
          {showLogs && (
            <div className="glass rounded-xl border border-primary/20 bg-primary/5 shadow-xl animate-in fade-in zoom-in-95 duration-300 overflow-hidden mt-6">
              <div className="flex items-center justify-between p-4 border-b border-primary/10 bg-primary/10">
                <div className="flex items-center space-x-4">
                  <div className="flex items-center space-x-2">
                    <Terminal className="w-5 h-5 text-primary" />
                    <h3 className="font-semibold text-foreground">Activity & Logs</h3>
                  </div>
                  <div className="flex bg-background/50 rounded-lg p-1 space-x-1 border border-border/50">
                    <button
                      onClick={() => setActiveTab('logs')}
                      className={cn("px-3 py-1 text-xs rounded-md transition-all font-medium", activeTab === 'logs' ? "bg-primary text-primary-foreground shadow" : "text-muted-foreground hover:text-foreground")}
                    >
                      Operator Logs
                    </button>
                    <button
                      onClick={() => setActiveTab('events')}
                      className={cn("px-3 py-1 text-xs rounded-md transition-all font-medium", activeTab === 'events' ? "bg-primary text-primary-foreground shadow" : "text-muted-foreground hover:text-foreground")}
                    >
                      K8s Events
                    </button>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <button 
                    onClick={activeTab === 'events' ? loadEvents : loadLogs} 
                    className="p-1.5 hover:bg-primary/20 rounded-md transition-colors text-primary"
                    title="Refresh"
                  >
                    <RefreshCw size={14} className={cn((loadingEvents || loadingLogs) && "animate-spin")} />
                  </button>
                  <button 
                    onClick={() => setShowLogs(false)} 
                    className="p-1.5 hover:bg-primary/20 rounded-md transition-colors text-muted-foreground"
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
              <div className="p-0 max-h-[400px] overflow-y-auto font-mono text-xs">
                {activeTab === 'events' && (
                  loadingEvents && events.length === 0 ? (
                    <div className="p-8 flex items-center justify-center text-muted-foreground">
                      <RefreshCw className="w-5 h-5 animate-spin mr-3 text-primary" />
                      <span>Fetching events from API server...</span>
                    </div>
                  ) : events.length === 0 ? (
                    <div className="p-12 flex flex-col items-center justify-center text-muted-foreground text-center">
                      <AlertCircle className="w-8 h-8 mb-3 opacity-20" />
                      <p>No events found for this resource.</p>
                      <p className="text-[10px] mt-1 opacity-50">Events are typically retained for 1 hour by Kubernetes.</p>
                    </div>
                  ) : (
                    <div className="divide-y divide-border/30">
                      {events.map((event: EventResponse, idx: number) => (
                        <div key={idx} className="p-3 hover:bg-background/40 transition-colors flex gap-4">
                          <div className="flex-shrink-0 w-24 text-muted-foreground/60 tabular-nums">
                            {event.age}
                          </div>
                          <div className={cn(
                            "flex-shrink-0 px-1.5 py-0.5 rounded text-[10px] h-fit font-bold uppercase",
                            event.type === 'Warning' ? "bg-amber-500/10 text-amber-500" : "bg-emerald-500/10 text-emerald-500"
                          )}>
                            {event.reason}
                          </div>
                          <div className="flex-grow text-foreground/90 leading-relaxed break-all">
                            {event.message}
                          </div>
                        </div>
                      ))}
                    </div>
                  )
                )}
                {activeTab === 'logs' && (
                  loadingLogs && logs.length === 0 ? (
                    <div className="p-8 flex items-center justify-center text-muted-foreground">
                      <RefreshCw className="w-5 h-5 animate-spin mr-3 text-primary" />
                      <span>Fetching operator logs...</span>
                    </div>
                  ) : logs.length === 0 ? (
                    <div className="p-12 flex flex-col items-center justify-center text-muted-foreground text-center">
                      <AlertCircle className="w-8 h-8 mb-3 opacity-20" />
                      <p>No reconciliation logs found for this resource yet.</p>
                    </div>
                  ) : (
                    <div className="bg-background/80 p-4 font-mono text-xs whitespace-pre-wrap rounded-b-xl leading-relaxed text-muted-foreground break-all">
                      {logs.map((log: LogResponse, idx: number) => (
                        <div key={idx} className="mb-2">
                          <span className="text-primary/70 mr-2">[{log.timestamp || 'sys'}]</span>
                          <span className={cn(
                            "px-1 py-0.5 rounded text-[10px] uppercase font-bold mr-2",
                            log.level === 'error' ? "bg-red-500/20 text-red-400" :
                            log.level === 'warn' ? "bg-amber-500/20 text-amber-400" :
                            "bg-blue-500/10 text-blue-400"
                          )}>{log.level}</span>
                          <span className="text-foreground/90">{log.message || log.raw}</span>
                        </div>
                      ))}
                    </div>
                  )
                )}
              </div>
            </div>
          )}
        </div>

        {/* Sidebar Column */}
        <div className="space-y-6">
          {/* Lifecycle Component */}
          <div className="glass rounded-xl p-6 border-border/50 sticky top-6">
            <h2 className="text-lg font-semibold border-b border-border/50 pb-4 mb-5">Lifecycle Status</h2>
            <LifecycleTimeline currentState={instance.state} />
          </div>
        </div>
      </div>
    </div>
  );
}

function DetailItem({ label, value, copyable }: { label: string; value: string; copyable?: boolean }) {
  return (
    <div className="flex flex-col group">
      <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold mb-1.5">{label}</span>
      <div className="flex items-center">
        <span className="text-foreground font-medium tracking-tight text-sm">{value}</span>
        {copyable && value !== 'None' && value !== 'Pending' && (
          <button 
            onClick={() => navigator.clipboard.writeText(value)}
            className="ml-2 p-1 text-muted-foreground hover:text-primary opacity-0 group-hover:opacity-100 transition-all rounded bg-card hover:bg-background"
            title="Copy to clipboard"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
          </button>
        )}
      </div>
    </div>
  );
}
