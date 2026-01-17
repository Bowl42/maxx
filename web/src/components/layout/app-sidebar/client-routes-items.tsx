import { NavLink, useLocation } from 'react-router-dom';
import {
  ClientIcon,
  allClientTypes,
  getClientName,
  getClientColor,
} from '@/components/icons/client-icons';
import { StreamingBadge } from '@/components/ui/streaming-badge';
import { MarqueeBackground } from '@/components/ui/marquee-background';
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
        <MarqueeBackground show={streamingCount > 0 && !isActive} color={color} opacity={0.5} />
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
