import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar';
import { NavProxyStatus } from '../nav-proxy-status';
import { SidebarRenderer } from './sidebar-renderer';
import { sidebarConfig } from './sidebar-config';
import { NavUser } from './nav-user';

export function AppSidebar() {
  return (
    <Sidebar collapsible="icon" className="border-border">
      <SidebarHeader className="h-[73px] border-b border-border justify-center">
        <NavProxyStatus />
      </SidebarHeader>

      <SidebarContent>
        <SidebarRenderer config={sidebarConfig} />
      </SidebarContent>

      <SidebarFooter className="border-t border-border">
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
