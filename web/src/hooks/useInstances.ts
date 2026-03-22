import { useState, useEffect, useCallback } from 'react';
import type { InstanceResponse, WatchEvent } from '../types/instance';
import { fetchInstances, fetchInstance, createEventSource } from '../api/client';

export const useInstances = () => {
  const [instances, setInstances] = useState<InstanceResponse[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  const loadInstances = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fetchInstances();
      // Ensure data is an array
      setInstances(Array.isArray(data) ? data : []);
      setError(null);
    } catch (err) {
      console.error('Initial fetch failed:', err);
      setError(err instanceof Error ? err : new Error('Failed to load'));
    } finally {
      setLoading(false);
    }
  }, []);

  const refetchInstance = useCallback(async (name: string, namespace: string) => {
    try {
      setLoading(true);
      const data = await fetchInstance(name, namespace);
      setInstances(prev => {
        const currentList = Array.isArray(prev) ? prev : [];
        const exists = currentList.find(i => i.name === data.name && i.namespace === data.namespace);
        
        if (exists) {
          return currentList.map(i => 
            (i.name === data.name && i.namespace === data.namespace) ? data : i
          );
        }
        return [...currentList, data];
      });
      setError(null);
    } catch (err) {
      console.error('Targeted fetch failed:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadInstances();

    const eventSource = createEventSource();
    
    eventSource.onopen = () => {
      console.log('SSE connection opened');
    };

    eventSource.onmessage = (event) => {
      try {
        const watchEvent: WatchEvent = JSON.parse(event.data);
        const { type, object } = watchEvent;
        
        setInstances(prev => {
          // Guard against prev being somehow undefined or null
          const currentList = Array.isArray(prev) ? prev : [];

          if (type === 'ADDED') {
            const exists = currentList.find(i => i.name === object.name && i.namespace === object.namespace);
            if (!exists) return [...currentList, object];
          } 
          else if (type === 'MODIFIED') {
            return currentList.map(i => 
              (i.name === object.name && i.namespace === object.namespace) ? object : i
            );
          } 
          else if (type === 'DELETED') {
            return currentList.filter(i => 
              !(i.name === object.name && i.namespace === object.namespace)
            );
          }
          return currentList;
        });
      } catch (err) {
        console.error('Error parsing SSE message:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('SSE connection error:', err);
      eventSource.close();
      // Optionally reconnect logic here
    };

    return () => {
      eventSource.close();
    };
  }, [loadInstances]);

  return { instances, loading, error, refetch: loadInstances, refetchInstance };
};
