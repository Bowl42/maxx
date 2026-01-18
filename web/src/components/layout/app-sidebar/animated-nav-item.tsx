import { NavLink, useLocation } from 'react-router-dom';
import { StreamingBadge } from '@/components/ui/streaming-badge';
import { MarqueeBackground } from '@/components/ui/marquee-background';
import { SidebarMenuBadge, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';
import type { ReactNode } from 'react';

interface AnimatedNavItemProps {
  /** The route path to navigate to */
  to: string;
  /** Function to check if the route is active */
  isActive: (pathname: string) => boolean;
  /** Tooltip text */
  tooltip: string;
  /** Icon element */
  icon: ReactNode;
  /** Label text */
  label: string;
  /** Streaming count for badge */
  streamingCount: number;
  /** Color for marquee and badge */
  color: string;
}

/**
 * Reusable navigation item with marquee background and streaming badge
 */
export function AnimatedNavItem({
  to,
  isActive: isActiveFn,
  tooltip,
  icon,
  label,
  streamingCount,
  color,
}: AnimatedNavItemProps) {
  const location = useLocation();
  const isActive = isActiveFn(location.pathname);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<NavLink to={to} />}
        isActive={isActive}
        tooltip={tooltip}
        className="relative overflow-hidden"
      >
        <MarqueeBackground show={streamingCount > 0} color={color} opacity={0.3} />
        {icon}
        <span>{label}</span>
      </SidebarMenuButton>
      <SidebarMenuBadge>
        <StreamingBadge count={streamingCount} color={color} />
      </SidebarMenuBadge>
    </SidebarMenuItem>
  );
}
