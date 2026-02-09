'use client';

import { Moon, Sun, Laptop, Languages, Sparkles, Gem } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useTheme } from '@/components/theme-provider';
import type { Theme } from '@/lib/theme';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from '@/components/ui/dropdown-menu';
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar';

export function NavUser() {
  const { isMobile } = useSidebar();
  const { t, i18n } = useTranslation();
  const { theme, setTheme } = useTheme();

  const user = {
    name: 'Maxx',
    avatar: '/logo.png',
  };

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={(props) => (
              <SidebarMenuButton
                {...props}
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground justify-center p-0!"
              >
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback className="rounded-lg">
                    {user.name.substring(0, 2).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
              </SidebarMenuButton>
            )}
          />
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] rounded-lg max-w-xs"
            side={isMobile ? 'bottom' : 'right'}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel>
                <div className="flex items-center gap-2 w-full">
                  <Avatar className="h-8 w-8 rounded-lg">
                    <AvatarImage src={user.avatar} alt={user.name} />
                    <AvatarFallback className="rounded-lg">
                      {user.name.substring(0, 2).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user.name}</span>
                    {/* <span className="text-xs truncate">{user.description}</span> */}
                  </div>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
            </DropdownMenuGroup>
            <DropdownMenuGroup>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  {theme === 'light' ? (
                    <Sun />
                  ) : theme === 'dark' ? (
                    <Moon />
                  ) : theme === 'hermes' || theme === 'tiffany' ? (
                    <Sparkles />
                  ) : (
                    <Laptop />
                  )}
                  <span>{t('nav.theme')}</span>
                </DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent>
                    <DropdownMenuRadioGroup value={theme} onValueChange={(v) => setTheme(v as Theme)}>
                      <DropdownMenuLabel className="text-xs text-muted-foreground">
                        {t('settings.themeDefault')}
                      </DropdownMenuLabel>
                      <DropdownMenuRadioItem value="light" closeOnClick>
                        <Sun />
                        <span>{t('settings.theme.light')}</span>
                      </DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value="dark" closeOnClick>
                        <Moon />
                        <span>{t('settings.theme.dark')}</span>
                      </DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value="system" closeOnClick>
                        <Laptop />
                        <span>{t('settings.theme.system')}</span>
                      </DropdownMenuRadioItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuLabel className="text-xs text-muted-foreground">
                        {t('settings.themeLuxury')}
                      </DropdownMenuLabel>
                      <DropdownMenuRadioItem value="hermes" closeOnClick>
                        <Sparkles className="text-orange-500" />
                        <span>{t('settings.theme.hermes')}</span>
                      </DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value="tiffany" closeOnClick>
                        <Gem className="text-cyan-500" />
                        <span>{t('settings.theme.tiffany')}</span>
                      </DropdownMenuRadioItem>
                    </DropdownMenuRadioGroup>
                  </DropdownMenuSubContent>
                </DropdownMenuPortal>
              </DropdownMenuSub>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Languages />
                  <span>{t('nav.language')}</span>
                </DropdownMenuSubTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuSubContent>
                    <DropdownMenuItem onClick={() => i18n.changeLanguage('en')}>
                      <span>{t('settings.languages.en')}</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => i18n.changeLanguage('zh')}>
                      <span>{t('settings.languages.zh')}</span>
                    </DropdownMenuItem>
                  </DropdownMenuSubContent>
                </DropdownMenuPortal>
              </DropdownMenuSub>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem
                render={
                  <a
                    href="https://github.com/awsl-project/maxx"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <svg
                      viewBox="0 0 24 24"
                      aria-hidden="true"
                      focusable="false"
                      className="size-4 fill-current"
                    >
                      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                    </svg>
                    GitHub
                  </a>
                }
              />
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
