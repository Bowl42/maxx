/**
 * Settings API Hooks
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';
import type { AntigravityGlobalSettings } from '@/lib/transport';

export const settingsKeys = {
  all: ['settings'] as const,
  detail: (key: string) => ['settings', key] as const,
  antigravityGlobal: ['settings', 'antigravity-global'] as const,
};

export function useSettings() {
  return useQuery({
    queryKey: settingsKeys.all,
    queryFn: () => getTransport().getSettings(),
  });
}

export function useSetting(key: string) {
  return useQuery({
    queryKey: settingsKeys.detail(key),
    queryFn: () => getTransport().getSetting(key),
    enabled: !!key,
  });
}

export function useUpdateSetting() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) =>
      getTransport().updateSetting(key, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.all });
    },
  });
}

export function useDeleteSetting() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (key: string) => getTransport().deleteSetting(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.all });
    },
  });
}

// ===== Antigravity Global Settings =====

export function useAntigravityGlobalSettings() {
  return useQuery({
    queryKey: settingsKeys.antigravityGlobal,
    queryFn: () => getTransport().getAntigravityGlobalSettings(),
  });
}

export function useUpdateAntigravityGlobalSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settings: AntigravityGlobalSettings) =>
      getTransport().updateAntigravityGlobalSettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.antigravityGlobal });
    },
  });
}

export function useResetAntigravityGlobalSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => getTransport().resetAntigravityGlobalSettings(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: settingsKeys.antigravityGlobal });
    },
  });
}
