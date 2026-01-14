/**
 * RetryConfig React Query Hooks
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransport, type RetryConfig, type CreateRetryConfigData } from '@/lib/transport';

// Query Keys
export const retryConfigKeys = {
  all: ['retryConfigs'] as const,
  lists: () => [...retryConfigKeys.all, 'list'] as const,
  list: () => [...retryConfigKeys.lists()] as const,
  details: () => [...retryConfigKeys.all, 'detail'] as const,
  detail: (id: number) => [...retryConfigKeys.details(), id] as const,
};

// 获取所有 RetryConfigs
export function useRetryConfigs() {
  return useQuery({
    queryKey: retryConfigKeys.list(),
    queryFn: () => getTransport().getRetryConfigs(),
  });
}

// 获取单个 RetryConfig
export function useRetryConfig(id: number) {
  return useQuery({
    queryKey: retryConfigKeys.detail(id),
    queryFn: () => getTransport().getRetryConfig(id),
    enabled: id > 0,
  });
}

// 创建 RetryConfig
export function useCreateRetryConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateRetryConfigData) => getTransport().createRetryConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: retryConfigKeys.lists() });
    },
  });
}

// 更新 RetryConfig
export function useUpdateRetryConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<RetryConfig> }) =>
      getTransport().updateRetryConfig(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: retryConfigKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: retryConfigKeys.lists() });
    },
  });
}

// 删除 RetryConfig
export function useDeleteRetryConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => getTransport().deleteRetryConfig(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: retryConfigKeys.lists() });
    },
  });
}
