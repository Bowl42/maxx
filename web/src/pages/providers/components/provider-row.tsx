import { ChevronRight, Globe, Mail, Server, Wand2, Activity } from 'lucide-react';
import { ClientIcon } from '@/components/icons/client-icons';
import { getProviderColor } from '@/lib/provider-colors';
import type { Provider, ProviderStats } from '@/lib/transport';
import { ANTIGRAVITY_COLOR } from '../types';
import { cn } from '@/lib/utils';

// 格式化 Token 数量
function formatTokens(count: number): string {
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toFixed(1)}M`;
  }
  if (count >= 1_000) {
    return `${(count / 1_000).toFixed(1)}K`;
  }
  return count.toString();
}

// 格式化成本 (微美元 → 美元)
function formatCost(microUsd: number): string {
  const usd = microUsd / 1_000_000;
  if (usd >= 1) {
    return `$${usd.toFixed(2)}`;
  }
  if (usd >= 0.01) {
    return `$${usd.toFixed(3)}`;
  }
  return `$${usd.toFixed(4)}`;
}

interface ProviderRowProps {
  provider: Provider;
  stats?: ProviderStats;
  streamingCount: number;
  onClick: () => void;
}

export function ProviderRow({ provider, stats, streamingCount, onClick }: ProviderRowProps) {
  const isAntigravity = provider.type === 'antigravity';
  const color = isAntigravity ? ANTIGRAVITY_COLOR : getProviderColor(provider.type);

  const getDisplayInfo = () => {
    if (isAntigravity) {
      return provider.config?.antigravity?.email || 'Unknown';
    }
    if (provider.config?.custom?.baseURL) return provider.config.custom.baseURL;
    for (const ct of provider.supportedClientTypes || []) {
      const url = provider.config?.custom?.clientBaseURL?.[ct];
      if (url) return url;
    }
    return 'Not configured';
  };

  return (
    <div
      onClick={onClick}
      className="group relative flex items-center gap-4 p-4 rounded-xl border border-border bg-surface-primary hover:border-accent/30 hover:shadow-card-hover cursor-pointer transition-all duration-200 overflow-hidden"
    >
      {/* Marquee 背景动画 (仅在有 streaming 请求时显示) */}
      {streamingCount > 0 && (
        <div
          className="absolute inset-0 animate-marquee pointer-events-none opacity-20"
          style={{ backgroundColor: `${color}15` }}
        />
      )}

      {/* Provider Icon */}
      <div
        className="relative z-10 w-12 h-12 rounded-xl flex items-center justify-center flex-shrink-0 shadow-sm border border-white/5"
        style={{ backgroundColor: `${color}15` }}
      >
        {isAntigravity ? (
          <Wand2 size={24} style={{ color }} />
        ) : (
          <Server size={24} style={{ color }} />
        )}
        {streamingCount > 0 && (
           <div className="absolute -top-1 -right-1">
              <span className="relative flex h-3 w-3">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-accent"></span>
              </span>
           </div>
        )}
      </div>

      {/* Provider Info */}
      <div className="relative z-10 flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <h3 className="text-base font-semibold text-text-primary truncate">{provider.name}</h3>
          <span
            className="px-2 py-0.5 rounded text-[10px] uppercase tracking-wide font-bold shrink-0"
            style={{ backgroundColor: `${color}15`, color }}
          >
            {isAntigravity ? 'Antigravity' : 'Custom'}
          </span>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-text-secondary truncate" title={getDisplayInfo()}>
          {isAntigravity ? <Mail size={12} className="opacity-70 shrink-0" /> : <Globe size={12} className="opacity-70 shrink-0" />}
          <span className="truncate">{getDisplayInfo()}</span>
        </div>
      </div>

      {/* Supported Clients - Fixed Width Column */}
      <div className="relative z-10 w-32 flex flex-col items-start gap-1 flex-shrink-0">
        <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium">Clients</span>
        <div className="flex items-center -space-x-1 hover:space-x-1 transition-all">
          {provider.supportedClientTypes?.length > 0 ? (
            provider.supportedClientTypes.map((ct) => (
              <div key={ct} className="relative z-0 hover:z-10 bg-surface-primary rounded-full p-0.5 border border-surface-primary transition-transform hover:scale-110">
                 <ClientIcon type={ct} size={16} />
              </div>
            ))
          ) : (
            <span className="text-xs text-text-muted">-</span>
          )}
        </div>
      </div>

      {/* Provider Stats */}
      <div className="relative z-10 flex items-center gap-2 bg-surface-secondary/30 rounded-lg p-1 border border-border/30">
        {stats && stats.totalRequests > 0 ? (
          <>
            {/* Success Rate */}
            <div className="flex flex-col items-center justify-center px-3 py-1 min-w-[70px]">
              <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium mb-0.5">Success</span>
              <span
                className={cn(
                  "font-mono font-bold text-sm",
                  stats.successRate >= 95 ? "text-emerald-400" :
                  stats.successRate >= 90 ? "text-blue-400" :
                  stats.successRate >= 80 ? "text-amber-400" : "text-red-400"
                )}
              >
                {stats.successRate.toFixed(1)}%
              </span>
            </div>
            
            <div className="w-px h-8 bg-border/40" />
            
            {/* Request Count */}
            <div className="flex flex-col items-center justify-center px-3 py-1 min-w-[70px]">
              <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium mb-0.5">Reqs</span>
              <span className="font-mono font-bold text-sm text-text-primary">{stats.totalRequests}</span>
            </div>

            <div className="w-px h-8 bg-border/40" />

            {/* Token Usage */}
            <div className="flex flex-col items-center justify-center px-3 py-1 min-w-[70px]">
              <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium mb-0.5">Tokens</span>
              <span className="font-mono font-bold text-sm text-blue-400">
                {formatTokens(stats.totalInputTokens + stats.totalOutputTokens)}
              </span>
            </div>

            <div className="w-px h-8 bg-border/40" />

            {/* Cost */}
            <div className="flex flex-col items-center justify-center px-3 py-1 min-w-[70px]">
              <span className="text-[10px] text-text-muted uppercase tracking-wider font-medium mb-0.5">Cost</span>
              <span className="font-mono font-bold text-sm text-purple-400">{formatCost(stats.totalCost)}</span>
            </div>
          </>
        ) : (
          <div className="px-6 py-3 flex items-center gap-2 text-text-muted/50">
             <Activity size={16} />
             <span className="text-xs font-medium">No activity yet</span>
          </div>
        )}
      </div>

      {/* Arrow */}
      <div className="relative z-10 pl-2">
         <div className="p-2 rounded-full text-text-muted hover:bg-surface-secondary hover:text-text-primary transition-colors">
            <ChevronRight size={18} />
         </div>
      </div>
    </div>
  );
}
