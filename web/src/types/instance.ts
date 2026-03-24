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
  storage?: {
    totalSize: number;
    rootVolume: {
      size: number;
      type: string;
      deviceName: string;
    };
    additionalVolumes?: {
      size: number;
      type: string;
      deviceName: string;
    }[];
  };
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

export interface InstanceCostData {
  instanceId: string;
  node: string;
  instanceType: string;
  region: string;
  dailyCost: number;
  monthlyCost: number;
  state: string;
}
