import { NavLink, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Activity } from 'lucide-react';
import { StreamingBadge } from '@/components/ui/streaming-badge';
import { MarqueeBackground } from '@/components/ui/marquee-background';
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
        <MarqueeBackground show={total > 0 && !isActive} color={color} opacity={0.4} />
        <Activity className="relative z-10" />
        <span className="relative z-10">{t('requests.title')}</span>
      </SidebarMenuButton>
      <SidebarMenuBadge>
        <StreamingBadge count={total} color={color} />
      </SidebarMenuBadge>
    </SidebarMenuItem>
  );
}
