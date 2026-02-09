import { Badge, Button } from '@/components/ui';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { ArrowLeft, RefreshCw } from 'lucide-react';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { statusVariant } from '../index';
import type { ProxyRequest, ClientType } from '@/lib/transport';
import { ClientIcon, getClientName, getClientColor } from '@/components/icons/client-icons';
import { formatDuration } from '@/lib/utils';
import { useTranslation } from 'react-i18next';
function formatCost(nanoUSD: number): string {
  if (nanoUSD === 0) return '-';
  // 向下取整到 6 位小数 (microUSD 精度)
  const usd = Math.floor(nanoUSD / 1000) / 1_000_000;
  return `$${usd.toFixed(6)}`;
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

interface RequestHeaderProps {
  request: ProxyRequest;
  onBack: () => void;
  onRecalculateCost?: () => void;
  isRecalculating?: boolean;
}

export function RequestHeader({
  request,
  onBack,
  onRecalculateCost,
  isRecalculating,
}: RequestHeaderProps) {
  const { t } = useTranslation();
  return (
    <div className="border-b border-border bg-card px-4 md:px-6 py-2 md:py-0 shrink-0">
      <div className="flex items-center gap-3 md:gap-6 w-full min-h-[56px] md:min-h-[73px]">
        {/* Left: Back + Main Info */}
        <div className="flex items-center gap-3 min-w-0 flex-1">
          <SidebarTrigger className="-ml-2" />
          <Button
            variant="ghost"
            size="icon"
            onClick={onBack}
            className="h-8 w-8 text-muted-foreground hover:text-foreground shrink-0"
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div
            className="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
            style={
              {
                backgroundColor: `${getClientColor(request.clientType as ClientType)}15`,
              } as React.CSSProperties
            }
          >
            <ClientIcon type={request.clientType as ClientType} size={24} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              <h2 className="text-lg font-semibold text-foreground tracking-tight leading-none truncate">
                {request.requestModel || t('requests.unknownModel')}
              </h2>
              <Badge variant={statusVariant[request.status]} className="capitalize shrink-0">
                {request.status.toLowerCase().replace('_', ' ')}
              </Badge>
            </div>
            <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground leading-none flex-wrap">
              <span className="font-mono bg-muted px-1.5 py-0.5 rounded">#{request.id}</span>
              <span>{getClientName(request.clientType as ClientType)}</span>
              <span>·</span>
              <span>{formatTime(request.startTime)}</span>
              {request.responseModel && request.responseModel !== request.requestModel && (
                <>
                  <span>·</span>
                  <span className="text-muted-foreground">
                    {t('requests.responseLabel')}{' '}
                    <span className="text-foreground">{request.responseModel}</span>
                  </span>
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Stats Grid - hidden on small, compact grid on medium, full row on xl */}
      <div className="hidden md:flex items-center gap-1 lg:gap-4 flex-wrap pb-2 xl:pb-0 -mt-1 xl:mt-0">
        <StatItem label="TTFT" value={request.ttft && request.ttft > 0 ? formatDuration(request.ttft) : '-'} />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <StatItem label={t('requests.duration')} value={request.duration ? formatDuration(request.duration) : '-'} valueClassName="text-foreground" />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <StatItem label={t('requests.input')} value={request.inputTokenCount > 0 ? request.inputTokenCount.toLocaleString() : '-'} />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <StatItem label={t('requests.output')} value={request.outputTokenCount > 0 ? request.outputTokenCount.toLocaleString() : '-'} valueClassName="text-foreground" />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <StatItem label={t('requests.cacheRead')} value={request.cacheReadCount > 0 ? request.cacheReadCount.toLocaleString() : '-'} valueClassName="text-violet-400" />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <StatItem label={t('requests.cacheWrite')} value={request.cacheWriteCount > 0 ? request.cacheWriteCount.toLocaleString() : '-'} valueClassName="text-amber-400" />
        <div className="w-px h-6 bg-border hidden lg:block" />
        <div className="text-center px-2 lg:px-3">
          <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">
            {t('requests.cost')}
          </div>
          <div className="text-sm font-mono font-medium text-blue-400 flex items-center gap-1">
            {formatCost(request.cost)}
            {onRecalculateCost && (
              <Tooltip>
                <TooltipTrigger
                  className="inline-flex items-center justify-center h-5 w-5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent disabled:opacity-50"
                  onClick={onRecalculateCost}
                  disabled={isRecalculating}
                >
                  <RefreshCw className={`h-3 w-3 ${isRecalculating ? 'animate-spin' : ''}`} />
                </TooltipTrigger>
                <TooltipContent>{t('requests.recalculateCost')}</TooltipContent>
              </Tooltip>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function StatItem({ label, value, valueClassName = 'text-muted-foreground' }: { label: string; value: string; valueClassName?: string }) {
  return (
    <div className="text-center px-2 lg:px-3">
      <div className="text-[10px] uppercase tracking-wider text-muted-foreground mb-0.5">
        {label}
      </div>
      <div className={`text-sm font-mono font-medium ${valueClassName}`}>
        {value}
      </div>
    </div>
  );
}
