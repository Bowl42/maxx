/**
 * ProxyRequest React Query Hooks
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { getTransport, type ProxyRequest, type ProxyUpstreamAttempt, type PaginationParams } from '@/lib/transport';

const transport = getTransport();

// Query Keys
export const requestKeys = {
  all: ['requests'] as const,
  lists: () => [...requestKeys.all, 'list'] as const,
  list: (params?: PaginationParams) => [...requestKeys.lists(), params] as const,
  details: () => [...requestKeys.all, 'detail'] as const,
  detail: (id: number) => [...requestKeys.details(), id] as const,
  attempts: (id: number) => [...requestKeys.detail(id), 'attempts'] as const,
};

// 获取所有 ProxyRequests
export function useProxyRequests(params?: PaginationParams) {
  return useQuery({
    queryKey: requestKeys.list(params),
    queryFn: () => transport.getProxyRequests(params),
  });
}

// 获取 ProxyRequests 总数
export function useProxyRequestsCount() {
  return useQuery({
    queryKey: ['requestsCount'] as const,
    queryFn: () => transport.getProxyRequestsCount(),
  });
}

// 获取单个 ProxyRequest
export function useProxyRequest(id: number) {
  return useQuery({
    queryKey: requestKeys.detail(id),
    queryFn: () => transport.getProxyRequest(id),
    enabled: id > 0,
  });
}

// 获取 ProxyRequest 的 Attempts
export function useProxyUpstreamAttempts(proxyRequestId: number) {
  return useQuery({
    queryKey: requestKeys.attempts(proxyRequestId),
    queryFn: () => transport.getProxyUpstreamAttempts(proxyRequestId),
    enabled: proxyRequestId > 0,
  });
}

// 订阅 ProxyRequest 实时更新
export function useProxyRequestUpdates() {
  const queryClient = useQueryClient();

  useEffect(() => {
    // 确保 Transport 已连接
    transport.connect().catch(console.error);

    // 订阅 ProxyRequest 更新事件
    const unsubscribeRequest = transport.subscribe<ProxyRequest>(
      'proxy_request_update',
      (updatedRequest) => {
        // 检查是否是新请求（通过详情缓存判断）
        const existingDetail = queryClient.getQueryData(requestKeys.detail(updatedRequest.id));
        const isNewRequest = !existingDetail;

        // 更新单个请求的缓存
        queryClient.setQueryData(
          requestKeys.detail(updatedRequest.id),
          updatedRequest
        );

        // 更新列表缓存（乐观更新）- 同时更新所有分页的缓存
        queryClient.setQueriesData<ProxyRequest[]>(
          { queryKey: requestKeys.lists() },
          (old) => {
            // 确保 old 是数组类型，防止错误更新其他缓存
            if (!old || !Array.isArray(old)) return old;
            const index = old.findIndex((r) => r.id === updatedRequest.id);
            if (index >= 0) {
              const newList = [...old];
              newList[index] = updatedRequest;
              return newList;
            }
            // 新请求只添加到第一页（offset=0 的页面）
            return [updatedRequest, ...old];
          }
        );

        // 新请求时乐观更新 count
        if (isNewRequest) {
          queryClient.setQueryData<number>(['requestsCount'], (old) => (old ?? 0) + 1);
        }
      }
    );

    // 订阅 ProxyUpstreamAttempt 更新事件
    const unsubscribeAttempt = transport.subscribe<ProxyUpstreamAttempt>(
      'proxy_upstream_attempt_update',
      (updatedAttempt) => {
        // 更新 Attempts 缓存
        queryClient.setQueryData<ProxyUpstreamAttempt[]>(
          requestKeys.attempts(updatedAttempt.proxyRequestID),
          (old) => {
            if (!old) return [updatedAttempt];
            const index = old.findIndex((a) => a.id === updatedAttempt.id);
            if (index >= 0) {
              const newList = [...old];
              newList[index] = updatedAttempt;
              return newList;
            }
            return [...old, updatedAttempt];
          }
        );
      }
    );

    return () => {
      unsubscribeRequest();
      unsubscribeAttempt();
    };
  }, [queryClient]);
}
