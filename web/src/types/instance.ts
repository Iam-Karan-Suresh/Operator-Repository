export interface InstanceResponse {
  name: string;
  namespace: string;
  instanceID: string;
  state: string;
  publicIP: string;
  privateIP: string;
  publicDNS: string;
  privateDNS: string;
  instanceType: string;
  amiId: string;
  region: string;
  availabilityZone?: string;
  tags?: Record<string, string>;
  createdAt: string;
  age: string;
}

export interface WatchEvent {
  type: 'ADDED' | 'MODIFIED' | 'DELETED';
  object: InstanceResponse;
}

export interface EventResponse {
  type: string;
  reason: string;
  message: string;
  time: string;
  age: string;
  object: string;
}

export interface LogResponse {
  timestamp: string;
  level: string;
  message: string;
  raw: string;
}
