import { useState, useEffect, useRef } from 'react';
import { Terminal, Trash2, Pause, Play, ArrowDown } from 'lucide-react';
import { useStreamingRequests } from '@/hooks/use-streaming';
import { ClientIcon, getClientName } from '@/components/icons/client-icons';
import type { ProxyRequest } from '@/lib/transport';

interface LogEntry {
  id: string;
  timestamp: Date;
  type: 'request' | 'response' | 'error' | 'info';
  clientType?: string;
  message: string;
  requestId?: string;
  model?: string;
  status?: string;
}

export function ConsolePage() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const { requests } = useStreamingRequests();
  const processedIds = useRef(new Set<string>());

  // Process streaming requests into log entries
  useEffect(() => {
    if (isPaused) return;

    requests.forEach((req: ProxyRequest) => {
      const key = `${req.requestID}-${req.status}`;
      if (processedIds.current.has(key)) return;
      processedIds.current.add(key);

      const entry: LogEntry = {
        id: key,
        timestamp: new Date(req.startTime),
        type: req.status === 'FAILED' ? 'error' : req.status === 'COMPLETED' ? 'response' : 'request',
        clientType: req.clientType,
        message: formatLogMessage(req),
        requestId: req.requestID,
        model: req.requestModel,
        status: req.status,
      };

      setLogs((prev) => [...prev.slice(-499), entry]);
    });
  }, [requests, isPaused]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  };

  const clearLogs = () => {
    setLogs([]);
    processedIds.current.clear();
  };

  const scrollToBottom = () => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    setAutoScroll(true);
  };

  return (
    <div className="flex flex-col h-full">
      <Header
        isPaused={isPaused}
        onTogglePause={() => setIsPaused(!isPaused)}
        onClear={clearLogs}
        logCount={logs.length}
      />

      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto bg-[#1a1a1a] font-mono text-sm"
      >
        {logs.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="p-4 space-y-1">
            {logs.map((log) => (
              <LogLine key={log.id} log={log} />
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>

      {!autoScroll && (
        <button
          onClick={scrollToBottom}
          className="absolute bottom-6 right-6 p-2 bg-accent text-white rounded-full shadow-lg hover:bg-accent-hover"
        >
          <ArrowDown size={20} />
        </button>
      )}
    </div>
  );
}

function Header({
  isPaused,
  onTogglePause,
  onClear,
  logCount,
}: {
  isPaused: boolean;
  onTogglePause: () => void;
  onClear: () => void;
  logCount: number;
}) {
  return (
    <div className="h-[73px] flex items-center justify-between p-lg border-b border-border bg-surface-primary">
      <div className="flex items-center gap-md">
        <div className="w-10 h-10 rounded-lg bg-emerald-400/10 flex items-center justify-center">
          <Terminal size={20} className="text-emerald-400" />
        </div>
        <div>
          <h1 className="text-headline font-semibold text-text-primary">Console</h1>
          <p className="text-caption text-text-secondary">{logCount} entries</p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onTogglePause}
          className={`btn flex items-center gap-2 ${isPaused ? 'bg-amber-500/20 text-amber-400' : 'bg-surface-secondary text-text-primary hover:bg-surface-hover'}`}
        >
          {isPaused ? <Play size={14} /> : <Pause size={14} />}
          {isPaused ? 'Resume' : 'Pause'}
        </button>
        <button
          onClick={onClear}
          className="btn bg-surface-secondary hover:bg-surface-hover text-text-primary flex items-center gap-2"
        >
          <Trash2 size={14} />
          Clear
        </button>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-text-muted">
      <Terminal size={48} className="mb-4 opacity-30" />
      <p>Waiting for requests...</p>
      <p className="text-xs mt-1">Logs will appear here in real-time</p>
    </div>
  );
}

function LogLine({ log }: { log: LogEntry }) {
  const timeStr = log.timestamp.toLocaleTimeString('en-US', { hour12: false });

  const typeColors = {
    request: 'text-blue-400',
    response: 'text-emerald-400',
    error: 'text-red-400',
    info: 'text-text-muted',
  };

  return (
    <div className="flex items-start gap-3 py-1 hover:bg-white/5 px-2 -mx-2 rounded">
      <span className="text-text-muted shrink-0">{timeStr}</span>
      {log.clientType && <ClientIcon type={log.clientType as any} size={16} className="shrink-0 mt-0.5" />}
      <span className={`shrink-0 uppercase text-xs font-medium ${typeColors[log.type]}`}>
        [{log.status || log.type.toUpperCase()}]
      </span>
      <span className="text-gray-300 break-all">{log.message}</span>
    </div>
  );
}

function formatLogMessage(req: ProxyRequest): string {
  const client = getClientName(req.clientType as any);
  const model = req.requestModel || 'unknown';

  switch (req.status) {
    case 'PENDING':
      return `${client} request started - model: ${model}`;
    case 'IN_PROGRESS':
      return `${client} streaming - model: ${model}`;
    case 'COMPLETED':
      const duration = req.duration ? `${(req.duration / 1e6).toFixed(0)}ms` : '-';
      const tokens = req.inputTokenCount + req.outputTokenCount;
      return `${client} completed - model: ${req.responseModel || model}, duration: ${duration}, tokens: ${tokens}`;
    case 'FAILED':
      return `${client} failed - model: ${model}, error: ${req.error || 'unknown'}`;
    default:
      return `${client} - model: ${model}`;
  }
}

export default ConsolePage;
