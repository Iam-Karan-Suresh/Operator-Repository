import { useState, useEffect, useCallback } from 'react';
import { fetchAllCosts, fetchInstanceCost } from '../api/client';
import type { InstanceCostData } from '../types/instance';

export function useAllCosts() {
  const [costs, setCosts] = useState<InstanceCostData[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchCosts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchAllCosts();
      setCosts(data || []);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch costs'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCosts();
    const interval = setInterval(fetchCosts, 30000); // refresh every 30s
    return () => clearInterval(interval);
  }, [fetchCosts]);

  return { costs, loading, error, refreshCosts: fetchCosts };
}

export function useInstanceCost(instanceId: string) {
  const [cost, setCost] = useState<InstanceCostData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchCost = useCallback(async () => {
    if (!instanceId) return;
    try {
      setLoading(true);
      setError(null);
      const data = await fetchInstanceCost(instanceId);
      setCost(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch instance cost'));
    } finally {
      setLoading(false);
    }
  }, [instanceId]);

  useEffect(() => {
    fetchCost();
  }, [fetchCost]);

  return { cost, loading, error, refreshCost: fetchCost };
}
