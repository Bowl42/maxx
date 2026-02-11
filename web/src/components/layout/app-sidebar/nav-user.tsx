'use client';

import { Moon, Sun, Laptop, Languages, Sparkles, Gem, Github, ChevronsUp } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useTheme } from '@/components/theme-provider';
import type { Theme } from '@/lib/theme';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
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
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar';

export function NavUser() {
  const { isMobile, state } = useSidebar();
  const { t, i18n } = useTranslation();
  const { theme, setTheme } = useTheme();
  const isCollapsed = !isMobile && state === 'collapsed';
  const currentLanguage = (i18n.resolvedLanguage || i18n.language || 'en').toLowerCase().startsWith('zh')
    ? 'zh'
    : 'en';
  const currentLanguageLabel =
    currentLanguage === 'zh' ? t('settings.languages.zh') : t('settings.languages.en');

  const handleToggleLanguage = () => {
    i18n.changeLanguage(currentLanguage === 'zh' ? 'en' : 'zh');
  };

  const user = {
    name: 'Maxx',
    avatar: '/logo.png',
  };

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <div
          className={cn(
            'flex items-center gap-2 rounded-xl border border-sidebar-border/70 bg-sidebar/70 p-1.5 backdrop-blur-sm',
            isCollapsed ? 'flex-col' : 'justify-between',
          )}
        >
          <a
            href="https://github.com/awsl-project/maxx"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            title="GitHub"
          >
            <Github className="h-4 w-4" />
          </a>

          <button
            type="button"
            onClick={handleToggleLanguage}
            title={`${t('nav.language')}: ${currentLanguageLabel}`}
            className={cn(
              'inline-flex items-center rounded-full border border-sidebar-border/70 bg-sidebar-accent/40 p-0.5 text-sidebar-foreground transition-colors hover:bg-sidebar-accent',
              isCollapsed ? 'h-8 w-8 justify-center' : 'h-8 px-1 gap-1',
            )}
          >
            {isCollapsed ? (
              <span className="text-[11px] font-semibold uppercase">
                {currentLanguage === 'zh' ? '中' : 'EN'}
              </span>
            ) : (
              <>
                <Languages className="h-3.5 w-3.5 text-sidebar-foreground/80" />
                <span className="inline-flex items-center rounded-full bg-sidebar/70 p-0.5">
                  <span
                    className={cn(
                      'rounded-full px-1.5 py-0.5 text-[10px] font-semibold uppercase transition-colors',
                      currentLanguage === 'zh'
                        ? 'bg-sidebar text-sidebar-foreground shadow-sm'
                        : 'text-sidebar-foreground/55',
                    )}
                  >
                    中
                  </span>
                  <span
                    className={cn(
                      'rounded-full px-1.5 py-0.5 text-[10px] font-semibold uppercase transition-colors',
                      currentLanguage === 'en'
                        ? 'bg-sidebar text-sidebar-foreground shadow-sm'
                        : 'text-sidebar-foreground/55',
                    )}
                  >
                    EN
                  </span>
                </span>
              </>
            )}
          </button>

          <DropdownMenu>
            <DropdownMenuTrigger
              render={(props) => (
                <button
                  {...props}
                  type="button"
                  title="Menu"
                  className={cn(
                    'inline-flex h-8 w-8 items-center justify-center rounded-lg text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                    props.className,
                  )}
                >
                  <ChevronsUp className="h-4 w-4" />
                </button>
              )}
            />
            <DropdownMenuContent
              className="w-64 rounded-lg max-w-xs"
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
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
