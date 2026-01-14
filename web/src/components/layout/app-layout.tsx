import { Outlet } from 'react-router-dom'
import { SidebarNav } from './sidebar-nav'
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar'
import { ForceProjectDialog } from '@/components/force-project-dialog'
import { usePendingSession } from '@/hooks/use-pending-session'
import { useSettings } from '@/hooks/queries'

export function AppLayout() {
  const { pendingSession, clearPendingSession } = usePendingSession()
  const { data: settings } = useSettings()

  const forceProjectEnabled = settings?.force_project_binding === 'true'
  const timeoutSeconds = parseInt(settings?.force_project_timeout || '30', 10)

  return (
    <SidebarProvider>
      <SidebarNav />
      <SidebarInset>
        <div className="@container/main h-full">
          <Outlet />
        </div>
      </SidebarInset>

      {/* Force Project Dialog - only show when enabled */}
      {forceProjectEnabled && (
        <ForceProjectDialog
          event={pendingSession}
          onClose={clearPendingSession}
          timeoutSeconds={timeoutSeconds}
        />
      )}
    </SidebarProvider>
  )
}
