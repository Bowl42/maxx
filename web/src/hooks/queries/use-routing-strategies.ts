/**
 * RoutingStrategy React Query Hooks
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransport, type RoutingStrategy, type CreateRoutingStrategyData } from '@/lib/transport';

// Query Keys
export const routingStrategyKeys = {
  all: ['routingStrategies'] as const,
  lists: () => [...routingStrategyKeys.all, 'list'] as const,
  list: () => [...routingStrategyKeys.lists()] as const,
  details: () => [...routingStrategyKeys.all, 'detail'] as const,
  detail: (id: number) => [...routingStrategyKeys.details(), id] as const,
};

// 获取所有 RoutingStrategies
export function useRoutingStrategies() {
  return useQuery({
    queryKey: routingStrategyKeys.list(),
    queryFn: () => getTransport().getRoutingStrategies(),
  });
}

// 获取单个 RoutingStrategy
export function useRoutingStrategy(id: number) {
  return useQuery({
    queryKey: routingStrategyKeys.detail(id),
    queryFn: () => getTransport().getRoutingStrategy(id),
    enabled: id > 0,
  });
}

// 创建 RoutingStrategy
export function useCreateRoutingStrategy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateRoutingStrategyData) => getTransport().createRoutingStrategy(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: routingStrategyKeys.lists() });
    },
  });
}

// 更新 RoutingStrategy
export function useUpdateRoutingStrategy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<RoutingStrategy> }) =>
      getTransport().updateRoutingStrategy(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: routingStrategyKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: routingStrategyKeys.lists() });
    },
  });
}

// 删除 RoutingStrategy
export function useDeleteRoutingStrategy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => getTransport().deleteRoutingStrategy(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: routingStrategyKeys.lists() });
    },
  });
}
