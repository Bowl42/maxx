import { useState, useMemo } from 'react';
import { Plus, Layers, Wand2, Server } from 'lucide-react';
import { useProviders, useAllProviderStats } from '@/hooks/queries';
import { useStreamingRequests } from '@/hooks/use-streaming';
import type { Provider } from '@/lib/transport';
import { ProviderRow } from './components/provider-row';
import { ProviderCreateFlow } from './components/provider-create-flow';
import { ProviderEditFlow } from './components/provider-edit-flow';

export function ProvidersPage() {
  const { data: providers, isLoading } = useProviders();
  const { data: providerStats = {} } = useAllProviderStats();
  const { countsByProvider } = useStreamingRequests();
  const [showCreateFlow, setShowCreateFlow] = useState(false);
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null);

  const groupedProviders = useMemo(() => {
    const antigravity: Provider[] = [];
    const custom: Provider[] = [];

    providers?.forEach(p => {
      if (p.type === 'antigravity') {
        antigravity.push(p);
      } else {
        custom.push(p);
      }
    });

    return { antigravity, custom };
  }, [providers]);

  // Show edit flow
  if (editingProvider) {
    return <ProviderEditFlow provider={editingProvider} onClose={() => setEditingProvider(null)} />;
  }

  // Show create flow
  if (showCreateFlow) {
    return <ProviderCreateFlow onClose={() => setShowCreateFlow(false)} />;
  }

  // Provider list
  return (
    <div className="flex flex-col h-full bg-background">
      <div className="h-[73px] flex items-center justify-between px-6 border-b border-border bg-surface-primary flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-accent/10 rounded-lg">
             <Layers size={20} className="text-accent" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-text-primary leading-tight">Providers</h2>
            <p className="text-xs text-text-secondary">{providers?.length || 0} configured</p>
          </div>
        </div>
        <button onClick={() => setShowCreateFlow(true)} className="btn btn-primary flex items-center gap-2">
          <Plus size={14} />
          <span>Add Provider</span>
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-text-muted">Loading...</div>
          </div>
        ) : providers?.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-text-muted">
            <Layers size={48} className="mb-4 opacity-50" />
            <p className="text-body">No providers configured</p>
            <p className="text-caption mt-2">Click "Add Provider" to create one</p>
            <button onClick={() => setShowCreateFlow(true)} className="btn btn-primary mt-6 flex items-center gap-2">
              <Plus size={14} />
              <span>Add Provider</span>
            </button>
          </div>
        ) : (
          <div className="space-y-8">
             {/* Antigravity Section */}
             {groupedProviders.antigravity.length > 0 && (
                <section className="space-y-3">
                   <div className="flex items-center gap-2 px-1">
                      <Wand2 size={16} className="text-purple-400" />
                      <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wider">Antigravity Cloud</h3>
                      <div className="h-px flex-1 bg-border/50 ml-2" />
                   </div>
                   <div className="space-y-3">
                      {groupedProviders.antigravity.map((provider) => (
                        <ProviderRow
                          key={provider.id}
                          provider={provider}
                          stats={providerStats[provider.id]}
                          streamingCount={countsByProvider.get(provider.id) || 0}
                          onClick={() => setEditingProvider(provider)}
                        />
                      ))}
                   </div>
                </section>
             )}

             {/* Custom Section */}
             {groupedProviders.custom.length > 0 && (
                <section className="space-y-3">
                   <div className="flex items-center gap-2 px-1">
                      <Server size={16} className="text-blue-400" />
                      <h3 className="text-sm font-semibold text-text-secondary uppercase tracking-wider">Custom Providers</h3>
                      <div className="h-px flex-1 bg-border/50 ml-2" />
                   </div>
                   <div className="space-y-3">
                      {groupedProviders.custom.map((provider) => (
                        <ProviderRow
                          key={provider.id}
                          provider={provider}
                          stats={providerStats[provider.id]}
                          streamingCount={countsByProvider.get(provider.id) || 0}
                          onClick={() => setEditingProvider(provider)}
                        />
                      ))}
                   </div>
                </section>
             )}
          </div>
        )}
      </div>
    </div>
  );
}