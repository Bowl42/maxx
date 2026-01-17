/**
 * Marquee Background 组件
 * 显示带有斜纹动画的背景效果，支持平滑淡入淡出
 */

interface MarqueeBackgroundProps {
  /** 是否显示动画 */
  show: boolean;
  /** 背景颜色 */
  color: string;
  /** 不透明度 (0-1) */
  opacity?: number;
  /** 自定义类名 */
  className?: string;
}

/**
 * Marquee Background
 * 特性：
 * - 条件显示时平滑淡入淡出
 * - 支持自定义颜色和不透明度
 * - 绝对定位，不影响布局
 */
export function MarqueeBackground({
  show,
  color,
  opacity = 0.4,
  className = '',
}: MarqueeBackgroundProps) {
  return (
    <div
      className={`absolute inset-0 animate-marquee pointer-events-none transition-opacity duration-300 ${className}`}
      style={{
        backgroundColor: color,
        opacity: show ? opacity : 0,
      }}
    />
  );
}
