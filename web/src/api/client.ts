import type { InstanceResponse, EventResponse, LogResponse } from '../types/instance';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

export const fetchInstances = async (): Promise<InstanceResponse[]> => {
  const response = await fetch(`${API_BASE_URL}/instances`);
  if (!response.ok) {
    throw new Error('Failed to fetch instances');
  }
  return response.json();
};

export const fetchInstance = async (name: string, namespace: string = 'default'): Promise<InstanceResponse> => {
  const response = await fetch(`${API_BASE_URL}/instances/${name}?namespace=${namespace}`);
  if (!response.ok) {
    throw new Error('Failed to fetch instance');
  }
  return response.json();
};

export const createEventSource = (): EventSource => {
  return new EventSource(`${API_BASE_URL}/instances/watch`);
};

export const fetchInstanceEvents = async (name: string, namespace: string = 'default'): Promise<EventResponse[]> => {
  const response = await fetch(`${API_BASE_URL}/instances/${name}/events?namespace=${namespace}`);
  if (!response.ok) {
    throw new Error('Failed to fetch events');
  }
  return response.json();
};

export const fetchInstanceLogs = async (name: string, namespace: string = 'default'): Promise<LogResponse[]> => {
  const response = await fetch(`${API_BASE_URL}/instances/${namespace}/${name}/logs`);
  if (!response.ok) {
    throw new Error('Failed to fetch logs');
  }
  return response.json();
};

export const fetchAllCosts = async (): Promise<import('../types/instance').InstanceCostData[]> => {
  const response = await fetch(`${API_BASE_URL}/cost/instances`);
  if (!response.ok) {
    throw new Error('Failed to fetch costs');
  }
  return response.json();
};

export const fetchInstanceCost = async (instanceId: string): Promise<import('../types/instance').InstanceCostData> => {
  const response = await fetch(`${API_BASE_URL}/cost/instances/${instanceId}`);
  if (!response.ok) {
    throw new Error('Failed to fetch instance cost');
  }
  return response.json();
};
