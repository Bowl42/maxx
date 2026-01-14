/**
 * Session React Query Hooks
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';

// Query Keys
export const sessionKeys = {
  all: ['sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: () => [...sessionKeys.lists()] as const,
};

// 获取所有 Sessions
export function useSessions() {
  return useQuery({
    queryKey: sessionKeys.list(),
    queryFn: () => getTransport().getSessions(),
  });
}

// 更新 Session 的 Project 绑定
export function useUpdateSessionProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sessionID, projectID }: { sessionID: string; projectID: number }) =>
      getTransport().updateSessionProject(sessionID, projectID),
    onSuccess: () => {
      // Invalidate sessions list
      queryClient.invalidateQueries({ queryKey: sessionKeys.all });
      // Also invalidate proxy requests as their projectID may have changed
      queryClient.invalidateQueries({ queryKey: ['proxyRequests'] });
    },
  });
}

// 拒绝 Session
export function useRejectSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (sessionID: string) => getTransport().rejectSession(sessionID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    },
  });
}
