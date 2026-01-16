import { NavLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

interface NavItem {
  to: string
  icon: LucideIcon
  labelKey: string
}

interface NavManagementProps {
  items: NavItem[]
  title?: string
}

export function NavManagement({ items, title }: NavManagementProps) {
  const location = useLocation()
  const { t } = useTranslation()

  return (
    <SidebarGroup>
      {title && <SidebarGroupLabel>{title}</SidebarGroupLabel>}
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map(item => {
            const Icon = item.icon
            const isActive = location.pathname.startsWith(item.to)

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
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
