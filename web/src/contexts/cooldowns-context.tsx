/**
 * Cooldowns Context
 * 提供共享的 Cooldowns 数据，减少重复请求
 */

import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';
import type { Cooldown } from '@/lib/transport';

interface CooldownsContextValue {
  cooldowns: Cooldown[];
  isLoading: boolean;
  getCooldownForProvider: (providerId: number, clientType?: string) => Cooldown | undefined;
  isProviderInCooldown: (providerId: number, clientType?: string) => boolean;
  getRemainingSeconds: (cooldown: Cooldown) => number;
  formatRemaining: (cooldown: Cooldown) => string;
  clearCooldown: (providerId: number) => void;
  isClearingCooldown: boolean;
}

const CooldownsContext = createContext<CooldownsContextValue | null>(null);

interface CooldownsProviderProps {
  children: ReactNode;
}

export function CooldownsProvider({ children }: CooldownsProviderProps) {
  const queryClient = useQueryClient();
  // Force re-render counter to trigger updates when cooldowns expire
  const [refreshKey, setRefreshKey] = useState(0);

  const {
    data: cooldowns = [],
    isLoading,
  } = useQuery({
    queryKey: ['cooldowns'],
    queryFn: () => getTransport().getCooldowns(),
    staleTime: 5000,
  });

  // Subscribe to cooldown_update WebSocket event
  useEffect(() => {
    const transport = getTransport();
    const unsubscribe = transport.subscribe('cooldown_update', () => {
      queryClient.invalidateQueries({ queryKey: ['cooldowns'] });
    });

    return () => {
      unsubscribe();
    };
  }, [queryClient]);

  // Mutation for clearing cooldown
  const clearCooldownMutation = useMutation({
    mutationFn: (providerId: number) => getTransport().clearCooldown(providerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cooldowns'] });
    },
  });

  // Setup timeouts for each cooldown to force re-render when they expire
  useEffect(() => {
    if (cooldowns.length === 0) {
      return;
    }

    const timeouts: number[] = [];

    cooldowns.forEach((cooldown) => {
      const until = new Date(cooldown.untilTime).getTime();
      const now = Date.now();
      const delay = until - now;

      if (delay > 0) {
        const timeout = setTimeout(() => {
          setRefreshKey((prev) => prev + 1);
        }, delay + 100);
        timeouts.push(timeout);
      }
    });

    return () => {
      timeouts.forEach((timeout) => clearTimeout(timeout));
    };
  }, [cooldowns]);

  const getCooldownForProvider = useCallback((providerId: number, clientType?: string) => {
    return cooldowns.find((cd: Cooldown) => {
      const matchesProvider = cd.providerID === providerId;
      const matchesClientType =
        cd.clientType === '' ||
        cd.clientType === 'all' ||
        (clientType && cd.clientType === clientType);

      if (!matchesProvider || !matchesClientType) {
        return false;
      }

      const untilTime =
        cd.untilTime || ((cd as unknown as Record<string, unknown>).until as string);
      if (!untilTime) {
        return false;
      }
      const until = new Date(untilTime).getTime();
      const now = Date.now();
      return until > now;
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cooldowns, refreshKey]);

  const isProviderInCooldown = useCallback((providerId: number, clientType?: string) => {
    return !!getCooldownForProvider(providerId, clientType);
  }, [getCooldownForProvider]);

  const getRemainingSeconds = useCallback((cooldown: Cooldown) => {
    const untilTime =
      cooldown.untilTime || ((cooldown as unknown as Record<string, unknown>).until as string);
    if (!untilTime) return 0;

    const until = new Date(untilTime);
    const now = new Date();
    const diff = until.getTime() - now.getTime();
    return Math.max(0, Math.floor(diff / 1000));
  }, []);

  const formatRemaining = useCallback((cooldown: Cooldown) => {
    const seconds = getRemainingSeconds(cooldown);

    if (Number.isNaN(seconds) || seconds === 0) return 'Expired';

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${String(hours).padStart(2, '0')}h ${String(minutes).padStart(2, '0')}m ${String(secs).padStart(2, '0')}s`;
    } else if (minutes > 0) {
      return `${String(minutes).padStart(2, '0')}m ${String(secs).padStart(2, '0')}s`;
    } else {
      return `${String(secs).padStart(2, '0')}s`;
    }
  }, [getRemainingSeconds]);

  const clearCooldown = useCallback((providerId: number) => {
    clearCooldownMutation.mutate(providerId);
  }, [clearCooldownMutation]);

  return (
    <CooldownsContext.Provider
      value={{
        cooldowns,
        isLoading,
        getCooldownForProvider,
        isProviderInCooldown,
        getRemainingSeconds,
        formatRemaining,
        clearCooldown,
        isClearingCooldown: clearCooldownMutation.isPending,
      }}
    >
      {children}
    </CooldownsContext.Provider>
  );
}

export function useCooldownsContext() {
  const context = useContext(CooldownsContext);
  if (!context) {
    throw new Error('useCooldownsContext must be used within CooldownsProvider');
  }
  return context;
}

// Optional hook that doesn't throw when used outside provider
export function useCooldownFromContext(providerId: number, clientType?: string): Cooldown | undefined {
  const context = useContext(CooldownsContext);
  return context?.getCooldownForProvider(providerId, clientType);
}
