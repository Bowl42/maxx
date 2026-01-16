import { NavLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

interface NavItem {
  to: string
  icon: LucideIcon
  labelKey: string
}

interface NavMainProps {
  items: NavItem[]
  children?: React.ReactNode
}

export function NavMain({ items, children }: NavMainProps) {
  const location = useLocation()
  const { t } = useTranslation()

  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map(item => {
            const Icon = item.icon
            const isActive =
              item.to === '/'
                ? location.pathname === '/'
                : location.pathname.startsWith(item.to)

            return (
              <SidebarMenuItem key={item.to}>
                <SidebarMenuButton
                  render={<NavLink to={item.to} />}
                  isActive={isActive}
                  tooltip={t(item.labelKey)}
                >
                  <Icon />
                  <span>{t(item.labelKey)}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          })}
          {children}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
