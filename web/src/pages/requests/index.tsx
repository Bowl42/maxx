import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useProxyRequests, useProxyRequestUpdates } from '@/hooks/queries';
import {
  Activity,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Loader2,
  CheckCircle,
  AlertTriangle,
} from 'lucide-react';
import type { ProxyRequest, ProxyRequestStatus } from '@/lib/transport';
import { ClientIcon } from '@/components/icons/client-icons';

const PAGE_SIZE = 50;

export const statusVariant: Record<ProxyRequestStatus, 'default' | 'success' | 'warning' | 'danger' | 'info'> = {
  PENDING: 'default',
  IN_PROGRESS: 'info',
  COMPLETED: 'success',
  FAILED: 'danger',
};

export function RequestsPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const { data: requests = [], isLoading, refetch } = useProxyRequests({ limit: PAGE_SIZE, offset: page * PAGE_SIZE });

  // Subscribe to real-time updates
  useProxyRequestUpdates();

  const total = requests.length;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="h-[73px] flex items-center justify-between p-lg border-b border-border bg-surface-primary flex-shrink-0">
        <div className="flex items-center gap-md">
          <Activity size={20} className="text-accent" />
          <h2 className="text-headline font-semibold text-text-primary">Requests</h2>
          <span className="text-caption text-text-secondary">
            {total} requests
          </span>
        </div>
        <div className="flex items-center gap-sm">
          <button
            onClick={() => refetch()}
            disabled={isLoading}
            className="btn bg-surface-secondary hover:bg-surface-hover text-text-primary flex items-center gap-xs"
          >
            <RefreshCw size={14} className={isLoading ? 'animate-spin' : ''} />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 flex flex-col overflow-hidden">
          {isLoading && requests.length === 0 ? (
            <div className="flex-1 flex items-center justify-center">
              <Loader2 className="w-8 h-8 animate-spin text-accent" />
            </div>
          ) : requests.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center text-text-muted">
              <Activity size={48} className="mb-md opacity-50" />
              <p className="text-body">No request logs yet</p>
              <p className="text-caption mt-xs">Requests will appear here when you start using the proxy</p>
            </div>
          ) : (
            <>
              {/* Pagination */}
              <div className="flex items-center justify-between px-md py-sm border-b border-border bg-surface-secondary/50">
                <span className="text-caption text-text-secondary">
                  {total > 0 ? (
                    <>
                      {page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, total)} of {total}
                    </>
                  ) : (
                    '0 items'
                  )}
                </span>
                <div className="flex items-center gap-xs">
                  <button
                    onClick={() => setPage((p) => Math.max(0, p - 1))}
                    disabled={page === 0}
                    className="p-xs rounded hover:bg-surface-hover text-text-secondary disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    <ChevronLeft size={16} />
                  </button>
                  <span className="text-caption text-text-secondary min-w-[60px] text-center">
                    {totalPages > 0 ? `${page + 1} / ${totalPages}` : '0 / 0'}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                    disabled={page >= totalPages - 1}
                    className="p-xs rounded hover:bg-surface-hover text-text-secondary disabled:opacity-30 disabled:cursor-not-allowed"
                  >
                    <ChevronRight size={16} />
                  </button>
                </div>
              </div>

              {/* Table */}
              <div className="flex-1 overflow-auto">
                <table className="w-full table-fixed border-collapse">
                  <thead className="bg-surface-secondary sticky top-0 z-10">
                    <tr>
                      <th className="w-[90px] text-left text-caption font-medium text-text-secondary p-md border-b border-border">Time</th>
                      <th className="w-[90px] text-left text-caption font-medium text-text-secondary p-md border-b border-border">Status</th>
                      <th className="w-[50px] text-left text-caption font-medium text-text-secondary p-md border-b border-border">Code</th>
                      <th className="w-[80px] text-left text-caption font-medium text-text-secondary p-md border-b border-border">Client</th>
                      <th className="w-[180px] text-left text-caption font-medium text-text-secondary p-md border-b border-border">Model</th>
                      <th className="w-[80px] text-right text-caption font-medium text-text-secondary p-md border-b border-border">Duration</th>
                      <th className="w-[70px] text-right text-caption font-medium text-text-secondary p-md border-b border-border">Cost</th>
                      <th className="w-[60px] text-center text-caption font-medium text-text-secondary p-md border-b border-border" title="Attempts">Att.</th>
                      <th className="w-[60px] text-right text-caption font-medium text-text-secondary p-md border-b border-border" title="Input Tokens">In</th>
                      <th className="w-[60px] text-right text-caption font-medium text-text-secondary p-md border-b border-border" title="Output Tokens">Out</th>
                      <th className="w-[60px] text-right text-caption font-medium text-text-secondary p-md border-b border-border" title="Cache Read">CacheR</th>
                      <th className="w-[60px] text-right text-caption font-medium text-text-secondary p-md border-b border-border" title="Cache Write">CacheW</th>
                    </tr>
                  </thead>
                  <tbody>
                    {requests.map((req) => (
                      <LogRow
                        key={req.id}
                        request={req}
                        onClick={() => navigate(`/requests/${req.id}`)}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// Request Status Badge Component
function RequestStatusBadge({ status }: { status: ProxyRequestStatus }) {
  const getStatusConfig = () => {
    switch (status) {
      case 'PENDING':
      case 'IN_PROGRESS':
        return {
          color: 'text-blue-400',
          bgColor: 'bg-blue-400/10',
          label: status === 'IN_PROGRESS' ? 'Streaming' : 'Pending',
          icon: (
            <span className="relative flex h-2 w-2 mr-1">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-400"></span>
            </span>
          ),
        };
      case 'COMPLETED':
        return {
          color: 'text-emerald-400',
          bgColor: 'bg-emerald-400/10 border border-emerald-400/20',
          label: 'Completed',
          icon: <CheckCircle size={10} className="mr-1" />,
        };
      case 'FAILED':
        return {
          color: 'text-white',
          bgColor: 'bg-rose-600 border border-rose-500 font-extrabold shadow-sm',
          label: 'Failed',
          icon: <AlertTriangle size={11} className="mr-1 stroke-[2.5px]" />,
        };
    }
  };

  const config = getStatusConfig();

  return (
    <div className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium ${config.color} ${config.bgColor}`}>
      {config.icon}
      <span>{config.label}</span>
    </div>
  );
}

// Token Cell Component
function TokenCell({ count, color }: { count: number; color: string }) {
  if (count === 0) {
    return <span className="text-caption text-text-muted font-mono">-</span>;
  }

  const formatTokens = (n: number) => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
    return n.toString();
  };

  return <span className={`text-caption font-mono ${color}`}>{formatTokens(count)}</span>;
}

// Cost Cell Component
function CostCell({ cost }: { cost: number }) {
  if (cost === 0) {
    return <span className="text-caption text-text-muted font-mono">-</span>;
  }

  const formatCost = (c: number) => {
    if (c < 0.01) return `$${c.toFixed(4)}`;
    if (c < 1) return `$${c.toFixed(3)}`;
    return `$${c.toFixed(2)}`;
  };

  const getCostColor = (c: number) => {
    if (c >= 0.10) return 'text-rose-400';
    if (c >= 0.01) return 'text-amber-400';
    return 'text-text-secondary';
  };

  return <span className={`text-caption font-mono ${getCostColor(cost)}`}>{formatCost(cost)}</span>;
}

// Log Row Component
function LogRow({
  request,
  onClick,
}: {
  request: ProxyRequest;
  onClick: () => void;
}) {
  const isPending = request.status === 'PENDING' || request.status === 'IN_PROGRESS';
  const isFailed = request.status === 'FAILED';

  // Live duration calculation for pending requests
  const [liveDuration, setLiveDuration] = useState<number | null>(null);

  useEffect(() => {
    if (!isPending) {
      setLiveDuration(null);
      return;
    }

    const startTime = new Date(request.startTime).getTime();
    const updateDuration = () => {
      const now = Date.now();
      setLiveDuration(now - startTime);
    };

    updateDuration();
    const interval = setInterval(updateDuration, 100);

    return () => clearInterval(interval);
  }, [isPending, request.startTime]);

  const formatDuration = (ns?: number | null) => {
    if (ns === undefined || ns === null) return '-';
    // If it's live duration (ms), convert directly
    if (isPending && liveDuration !== null) {
      return `${(liveDuration / 1000).toFixed(1)}s`;
    }
    // If it's stored duration (nanoseconds), convert
    const ms = ns / 1_000_000;
    if (ms < 1000) return `${ms.toFixed(0)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString();
  };

  // Display duration
  const displayDuration = isPending ? liveDuration : request.duration;

  // Duration color
  const durationColor = isPending
    ? 'text-white font-bold'
    : (displayDuration && displayDuration / 1_000_000 > 5000)
      ? 'text-amber-400'
      : 'text-text-secondary';

  // Row status style
  let rowStatusStyle = 'border-l-[3px] border-l-transparent';
  if (isPending) {
    rowStatusStyle = 'bg-accent/20 border-l-[3px] border-l-accent shadow-inner animate-pulse';
  } else if (isFailed) {
    rowStatusStyle = 'border-l-[3px] border-l-rose-500/80';
  }

  // Get HTTP status code from responseInfo
  const statusCode = request.responseInfo?.status;

  return (
    <tr
      onClick={onClick}
      className={`
        cursor-pointer border-b border-border/50 transition-all duration-300
        ${rowStatusStyle}
        ${!isPending ? 'hover:bg-surface-hover' : ''}
      `}
    >
      {/* Time */}
      <td className="p-md">
        <span className="text-caption text-text-muted">
          {formatTime(request.startTime || request.createdAt)}
        </span>
      </td>
      {/* Status */}
      <td className="p-md">
        <RequestStatusBadge status={request.status} />
      </td>
      {/* Code */}
      <td className="p-md">
        <span className={`text-caption font-mono ${isFailed ? 'text-rose-400' : request.status === 'COMPLETED' ? 'text-emerald-400' : 'text-text-muted'}`}>
          {statusCode && statusCode > 0 ? statusCode : '-'}
        </span>
      </td>
      {/* Client */}
      <td className="p-md">
        <div className="flex items-center gap-2">
          <ClientIcon type={request.clientType} size={16} />
          <span className="text-caption text-text-secondary capitalize">
            {request.clientType}
          </span>
        </div>
      </td>
      {/* Model */}
      <td className="p-md">
        <div className="flex flex-col">
          <span className="text-caption text-text-secondary truncate">
            {request.requestModel || '-'}
          </span>
          {request.responseModel && request.responseModel !== request.requestModel && (
            <span className="text-[10px] text-text-muted truncate">
              → {request.responseModel}
            </span>
          )}
        </div>
      </td>
      {/* Duration */}
      <td className="p-md text-right">
        <span className={`text-caption font-mono ${durationColor}`}>
          {formatDuration(displayDuration)}
        </span>
      </td>
      {/* Cost */}
      <td className="p-md text-right">
        <CostCell cost={request.cost} />
      </td>
      {/* Attempts */}
      <td className="p-md text-center">
        <span className={`text-caption font-mono ${request.proxyUpstreamAttemptCount > 1 ? 'text-warning' : 'text-text-muted'}`}>
          {request.proxyUpstreamAttemptCount || 1}
        </span>
      </td>
      {/* Input tokens */}
      <td className="p-md text-right">
        <TokenCell count={request.inputTokenCount} color="text-sky-400" />
      </td>
      {/* Output tokens */}
      <td className="p-md text-right">
        <TokenCell count={request.outputTokenCount} color="text-emerald-400" />
      </td>
      {/* Cache Read */}
      <td className="p-md text-right">
        <TokenCell count={request.cacheReadCount} color="text-violet-400" />
      </td>
      {/* Cache Write */}
      <td className="p-md text-right">
        <TokenCell count={request.cacheWriteCount} color="text-amber-400" />
      </td>
    </tr>
  );
}

export default RequestsPage;
