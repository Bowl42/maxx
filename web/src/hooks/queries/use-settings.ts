import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';

const transport = getTransport();

export interface SystemSetting {
  key: string;
  value: string;
}

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: () => transport.getSettings(),
  });
}

export function useSetting(key: string) {
  return useQuery({
    queryKey: ['settings', key],
    queryFn: () => transport.getSetting(key),
    enabled: !!key,
  });
}

export function useUpdateSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: string }) =>
      transport.updateSetting(key, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useDeleteSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => transport.deleteSetting(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}
