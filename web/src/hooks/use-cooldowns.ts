import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';
import type { Cooldown } from '@/lib/transport';
import { useEffect } from 'react';

export function useCooldowns() {
  const queryClient = useQueryClient();

  const {
    data: cooldowns = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: ['cooldowns'],
    queryFn: () => getTransport().getCooldowns(),
    refetchInterval: 5000, // Refetch every 5 seconds from server
    staleTime: 3000,
  });

  // Mutation for clearing cooldown
  const clearCooldownMutation = useMutation({
    mutationFn: (providerId: number) => getTransport().clearCooldown(providerId),
    onSuccess: () => {
      // Invalidate and refetch cooldowns after successful deletion
      queryClient.invalidateQueries({ queryKey: ['cooldowns'] });
    },
  });

  // Setup timeouts for each cooldown to invalidate when they expire
  useEffect(() => {
    if (cooldowns.length === 0) {
      return;
    }

    const timeouts: number[] = [];

    cooldowns.forEach((cooldown) => {
      const until = new Date(cooldown.untilTime).getTime();
      const now = Date.now();
      const delay = until - now;

      // If cooldown will expire in the future, set a timeout
      if (delay > 0) {
        const timeout = setTimeout(() => {
          // Invalidate query when cooldown expires
          queryClient.invalidateQueries({ queryKey: ['cooldowns'] });
        }, delay);
        timeouts.push(timeout);
      }
    });

    return () => {
      // Clear all timeouts on cleanup
      timeouts.forEach(timeout => clearTimeout(timeout));
    };
  }, [cooldowns, queryClient]);

  // Helper to get cooldown for a specific provider
  const getCooldownForProvider = (providerId: number, clientType?: string) => {
    return cooldowns.find(
      (cd: Cooldown) =>
        cd.providerID === providerId &&
        (cd.clientType === '' || cd.clientType === 'all' || (clientType && cd.clientType === clientType))
    );
  };

  // Helper to check if provider is in cooldown
  const isProviderInCooldown = (providerId: number, clientType?: string) => {
    return !!getCooldownForProvider(providerId, clientType);
  };

  // Helper to get remaining time as seconds
  const getRemainingSeconds = (cooldown: Cooldown) => {
    // Handle both 'untilTime' and 'until' field names for backward compatibility
    const untilTime = cooldown.untilTime || (cooldown as Record<string, unknown>).until as string;
    if (!untilTime) return 0;

    const until = new Date(untilTime);
    const now = new Date();
    const diff = until.getTime() - now.getTime();
    return Math.max(0, Math.floor(diff / 1000));
  };

  // Helper to format remaining time
  const formatRemaining = (cooldown: Cooldown) => {
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
  };

  // Helper to clear cooldown
  const clearCooldown = (providerId: number) => {
    clearCooldownMutation.mutate(providerId);
  };

  return {
    cooldowns,
    isLoading,
    error,
    getCooldownForProvider,
    isProviderInCooldown,
    getRemainingSeconds,
    formatRemaining,
    clearCooldown,
    isClearingCooldown: clearCooldownMutation.isPending,
  };
}
