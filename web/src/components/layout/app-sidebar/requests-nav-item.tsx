import { NavLink, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Activity } from 'lucide-react';
import { StreamingBadge } from '@/components/ui/streaming-badge';
import { useStreamingRequests } from '@/hooks/use-streaming';
import { SidebarMenuBadge, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar';

/**
 * Requests navigation item with streaming badge and marquee animation
 */
export function RequestsNavItem() {
  const location = useLocation();
  const { total } = useStreamingRequests();
  const { t } = useTranslation();
  const isActive =
    location.pathname === '/requests' || location.pathname.startsWith('/requests/');
  const color = 'var(--color-success)'; // emerald-500

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<NavLink to="/requests" />}
        isActive={isActive}
        tooltip={t('requests.title')}
        className="relative"
      >
        {/* Marquee 背景动画 (仅在有 streaming 请求且未激活时显示) */}
        {total > 0 && !isActive && (
          <div
            className="absolute inset-0 animate-marquee pointer-events-none opacity-40"
            style={{ backgroundColor: color }}
          />
        )}
        <Activity className="relative z-10" />
        <span className="relative z-10">{t('requests.title')}</span>
      </SidebarMenuButton>
      {total > 0 && (
        <SidebarMenuBadge>
          <StreamingBadge count={total} color={color} />
        </SidebarMenuBadge>
      )}
    </SidebarMenuItem>
  );
}
