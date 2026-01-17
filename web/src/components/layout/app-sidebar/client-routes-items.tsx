import { NavLink, useLocation } from 'react-router-dom';
import {
  ClientIcon,
  allClientTypes,
  getClientName,
  getClientColor,
} from '@/components/icons/client-icons';
import { StreamingBadge } from '@/components/ui/streaming-badge';
import { useStreamingRequests } from '@/hooks/use-streaming';
import type { ClientType } from '@/lib/transport';
import { SidebarMenuButton, SidebarMenuItem, SidebarMenuBadge } from '@/components/ui/sidebar';

function ClientNavItem({
  clientType,
  streamingCount
}: {
  clientType: ClientType;
  streamingCount: number;
}) {
  const location = useLocation();
  const color = getClientColor(clientType);
  const clientName = getClientName(clientType);
  const isActive = location.pathname === `/routes/${clientType}`;

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<NavLink to={`/routes/${clientType}`} />}
        isActive={isActive}
        tooltip={clientName}
        className="relative overflow-hidden"
      >
        {/* Marquee 背景动画 (仅在有 streaming 请求且未激活时显示) */}
        {streamingCount > 0 && !isActive && (
          <div
            className="absolute inset-0 animate-marquee pointer-events-none opacity-50"
            style={{ backgroundColor: color }}
          />
        )}
        <ClientIcon type={clientType} size={18} className="relative z-10" />
        <span className="relative z-10">{clientName}</span>
      </SidebarMenuButton>
      <SidebarMenuBadge>
        <StreamingBadge count={streamingCount} color={color} />
      </SidebarMenuBadge>
    </SidebarMenuItem>
  );
}

/**
 * Renders all client route items dynamically
 */
export function ClientRoutesItems() {
  const { countsByClient } = useStreamingRequests();

  return (
    <>
      {allClientTypes.map((clientType) => (
        <ClientNavItem
          key={clientType}
          clientType={clientType}
          streamingCount={countsByClient.get(clientType) || 0}
        />
      ))}
    </>
  );
}
