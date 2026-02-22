import type { QueryClient } from '@tanstack/react-query';
import { getTransport } from '@/lib/transport';

type Unsubscribe = () => void;

let transportUnsubscribe: Unsubscribe | null = null;
const queryClients = new Set<QueryClient>();

export function subscribeCooldownUpdates(queryClient: QueryClient): Unsubscribe {
  queryClients.add(queryClient);

  if (!transportUnsubscribe) {
    const transport = getTransport();
    transportUnsubscribe = transport.subscribe('cooldown_update', () => {
      for (const qc of queryClients) {
        qc.invalidateQueries({ queryKey: ['cooldowns'] });
      }
    });
  }

  return () => {
    queryClients.delete(queryClient);
    if (queryClients.size === 0 && transportUnsubscribe) {
      transportUnsubscribe();
      transportUnsubscribe = null;
    }
  };
}

