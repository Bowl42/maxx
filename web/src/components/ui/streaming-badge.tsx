/**
 * Streaming Badge 组件
 * 显示实时活动请求数，带延迟消失效果
 */

import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';

interface StreamingBadgeProps {
  /** 当前计数 */
  count: number;
  /** 徽章颜色 (用于边框和发光效果) */
  color?: string;
  /** 自定义类名 */
  className?: string;
  /** 延迟消失时间 (ms)，默认 1000 */
  hideDelay?: number;
}

/**
 * Streaming Badge
 * 特性：
 * - 计数 > 0 时立即显示
 * - 计数 = 0 时延迟隐藏，避免频繁闪烁
 * - 带脉冲动画和彩色发光效果
 */
export function StreamingBadge({
  count,
  color = '#0078D4',
  className,
  hideDelay = 1000,
}: StreamingBadgeProps) {
  // 使用 count 作为初始值，当 count > 0 时直接显示
  const [displayCount, setDisplayCount] = useState(count > 0 ? count : 0);

  // Effect 1: 处理 count 变化 - 当 count > 0 时立即显示
  useEffect(() => {
    if (count > 0) {
      setDisplayCount(count);
    }
  }, [count]);

  // Effect 2: 处理延迟隐藏 - 当 count = 0 且 displayCount > 0 时延迟隐藏
  useEffect(() => {
    if (count === 0 && displayCount > 0) {
      const timer = setTimeout(() => {
        setDisplayCount(0);
      }, hideDelay);

      return () => clearTimeout(timer);
    }
  }, [count, displayCount, hideDelay]);

  // 不显示时返回 null
  if (displayCount === 0) {
    return null;
  }

  return (
    <span
      className={cn(
        'px-1 rounded-sm text-xs font-extrabold animate-pulse-soft shadow-md text-center bg-secondary border-2',
        className,
      )}
      style={{
        borderColor: color,
        boxShadow: `0 0 10px ${color}60`,
      }}
    >
      {displayCount}
    </span>
  );
}
