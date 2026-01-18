import { useProxyStatus } from '@/hooks/queries';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarTrigger,
} from '@/components/ui/sidebar';
import { NavProxyStatus } from '../nav-proxy-status';
import { ThemeToggle } from '@/components/theme-toggle';
import { SidebarRenderer } from './sidebar-renderer';
import { sidebarConfig } from './sidebar-config';

export function AppSidebar() {
  const { data: proxyStatus } = useProxyStatus();
  const versionDisplay = proxyStatus?.version ?? '...';

  return (
    <Sidebar collapsible="icon" className="border-border">
      <SidebarHeader>
        <NavProxyStatus />
      </SidebarHeader>

      <SidebarContent>
        <SidebarRenderer config={sidebarConfig} />
      </SidebarContent>

      <SidebarFooter>
        <p className="text-caption text-muted-foreground group-data-[collapsible=icon]:hidden mb-2">
          {versionDisplay}
        </p>
        <div className="flex items-center gap-2 group-data-[collapsible=icon]:flex-col group-data-[collapsible=icon]:w-full group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:items-stretch">
          <SidebarTrigger />
          <ThemeToggle />
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
