import { useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { Snowflake, Clock, AlertCircle, Server, Wifi, Zap, Ban, HelpCircle, X } from 'lucide-react';
import type { Cooldown, CooldownReason } from '@/lib/transport/types';

interface CooldownDetailsDialogProps {
  cooldown: Cooldown | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onClear: () => void;
  isClearing: boolean;
  onDisable: () => void;
  isDisabling: boolean;
}

// Reason 中文说明和图标
const REASON_INFO: Record<CooldownReason, { label: string; description: string; icon: typeof Server }> = {
  server_error: {
    label: '服务器错误',
    description: '上游服务器返回 5xx 错误，系统自动进入冷却保护',
    icon: Server,
  },
  network_error: {
    label: '网络错误',
    description: '无法连接到上游服务器，可能是网络故障或服务器宕机',
    icon: Wifi,
  },
  quota_exhausted: {
    label: '配额耗尽',
    description: 'API 配额已用完，等待配额重置',
    icon: AlertCircle,
  },
  rate_limit_exceeded: {
    label: '速率限制',
    description: '请求速率超过限制，触发了速率保护',
    icon: Zap,
  },
  concurrent_limit: {
    label: '并发限制',
    description: '并发请求数超过限制',
    icon: Ban,
  },
  unknown: {
    label: '未知原因',
    description: '因未知原因进入冷却状态',
    icon: HelpCircle,
  },
};

export function CooldownDetailsDialog({
  cooldown,
  open,
  onOpenChange,
  onClear,
  isClearing,
  onDisable,
  isDisabling,
}: CooldownDetailsDialogProps) {
  // Handle ESC key
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      onOpenChange(false);
    }
  }, [onOpenChange]);

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
      return () => {
        document.removeEventListener('keydown', handleKeyDown);
        document.body.style.overflow = '';
      };
    }
  }, [open, handleKeyDown]);

  if (!open || !cooldown) return null;

  const reasonInfo = REASON_INFO[cooldown.reason] || REASON_INFO.unknown;
  const Icon = reasonInfo.icon;

  const formatUntilTime = (until: string) => {
    const date = new Date(until);
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  };

  // 使用项目已有的 dialog-overlay 和 dialog-content 全局样式
  return createPortal(
    <>
      {/* Overlay - 使用全局 .dialog-overlay 类 */}
      <div
        className="dialog-overlay"
        onClick={() => onOpenChange(false)}
        style={{ zIndex: 99998 }}
      />

      {/* Content - 使用全局 .dialog-content 类 */}
      <div
        className="dialog-content"
        style={{
          zIndex: 99999,
          width: '100%',
          maxWidth: '28rem',
          padding: 0,
          background: 'var(--color-surface-primary)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Decorative top gradient */}
        <div
          className="h-1 rounded-t-lg"
          style={{ background: 'linear-gradient(to right, #22d3ee, #3b82f6, #22d3ee)' }}
        />

        {/* Header */}
        <div className="px-6 py-5 flex items-center justify-between border-b border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-cyan-900/40 text-cyan-400 border border-cyan-800">
              <Snowflake size={22} className="animate-spin-slow" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-text-primary">冷却详情</h2>
              <p className="text-xs text-text-muted">Provider Cooldown Status</p>
            </div>
          </div>
          <button
            onClick={() => onOpenChange(false)}
            className="p-2 rounded-full hover:bg-surface-hover text-text-muted hover:text-text-primary transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-5">
          {/* Provider Info Card */}
          <div className="flex items-center gap-4 p-4 rounded-xl bg-surface-secondary border border-border">
            <div className="flex-1 min-w-0">
              <div className="text-xs font-medium text-text-muted uppercase tracking-wider mb-0.5">Provider</div>
              <div className="text-base font-semibold text-text-primary truncate">
                {cooldown.providerName || `Provider #${cooldown.providerID}`}
              </div>
            </div>
            {cooldown.clientType && (
              <div className="text-right flex-shrink-0">
                <div className="text-xs font-medium text-text-muted uppercase tracking-wider mb-0.5">Client</div>
                <div className="px-2 py-1 rounded text-xs font-mono font-medium bg-surface-hover text-text-secondary">
                  {cooldown.clientType}
                </div>
              </div>
            )}
          </div>

          {/* Reason Section */}
          <div className="relative overflow-hidden rounded-xl border border-cyan-800 bg-cyan-900/30">
            <div className="absolute top-0 left-0 w-1.5 h-full bg-cyan-400" />
            <div className="p-4 pl-5 flex gap-4">
              <div className="mt-0.5 flex-shrink-0 text-cyan-400">
                <Icon size={24} />
              </div>
              <div>
                <h3 className="font-semibold text-cyan-100 mb-1">{reasonInfo.label}</h3>
                <p className="text-sm text-cyan-300/80 leading-relaxed">
                  {reasonInfo.description}
                </p>
              </div>
            </div>
          </div>

          {/* Timing Grid */}
          <div className="grid grid-cols-2 gap-4">
            <div className="p-4 rounded-xl bg-surface-secondary border border-border">
              <div className="flex items-center gap-2 text-text-muted mb-2">
                <Clock size={14} />
                <span className="text-xs font-medium uppercase tracking-wider">恢复时间</span>
              </div>
              <div className="font-mono text-sm font-semibold text-text-secondary">
                {formatUntilTime(cooldown.until).split(' ')[0]}
                <br />
                {formatUntilTime(cooldown.until).split(' ')[1]}
              </div>
            </div>

            <div className="p-4 rounded-xl bg-cyan-900/30 border border-cyan-800">
              <div className="flex items-center gap-2 text-cyan-400 mb-2">
                <Snowflake size={14} className="animate-pulse" />
                <span className="text-xs font-medium uppercase tracking-wider">剩余冻结</span>
              </div>
              <div className="font-mono text-xl font-bold text-cyan-400">
                {cooldown.remaining}
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="pt-2 space-y-3">
            <button
              onClick={onClear}
              disabled={isClearing || isDisabling}
              className="w-full rounded-xl px-4 py-3 text-white font-semibold shadow-lg transition-all hover:brightness-110 disabled:opacity-50 disabled:cursor-not-allowed"
              style={{
                background: 'linear-gradient(to right, #06b6d4, #2563eb)',
                boxShadow: '0 10px 15px -3px rgba(6, 182, 212, 0.25)',
              }}
            >
              <div className="flex items-center justify-center gap-2">
                {isClearing ? (
                  <>
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                    <span>正在解冻...</span>
                  </>
                ) : (
                  <>
                    <Zap size={18} />
                    <span>立即解冻</span>
                  </>
                )}
              </div>
            </button>

            <button
              onClick={onDisable}
              disabled={isDisabling || isClearing}
              className="w-full rounded-xl px-4 py-3 font-semibold shadow transition-all hover:brightness-95 disabled:opacity-50 disabled:cursor-not-allowed"
              style={{
                background: 'var(--color-surface-secondary)',
                color: 'var(--color-warning)',
                border: '1px solid var(--color-border)',
              }}
            >
              <div className="flex items-center justify-center gap-2">
                {isDisabling ? (
                  <>
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-current/30 border-t-current" />
                    <span>正在禁用...</span>
                  </>
                ) : (
                  <>
                    <Ban size={18} />
                    <span>禁用此路由</span>
                  </>
                )}
              </div>
            </button>

            <p className="text-center text-xs text-text-muted">
              强制解冻可能会导致请求再次失败，禁用路由可避免持续失败
            </p>
          </div>
        </div>
      </div>
    </>,
    document.body
  );
}
