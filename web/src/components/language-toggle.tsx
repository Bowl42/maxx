import * as React from 'react';
import { Languages, Check } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from './ui';
import { cn } from '@/lib/utils';

const LANGUAGES = [
  { code: 'en', name: 'English', nativeName: 'English' },
  { code: 'zh', name: 'Chinese', nativeName: '中文' },
] as const;

export function LanguageToggle() {
  const { i18n } = useTranslation();
  const currentLanguage = LANGUAGES.find((lang) => lang.code === i18n.language) || LANGUAGES[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={(props) => (
          <Button
            {...props}
            title={`Current language: ${currentLanguage.nativeName}`}
            variant="ghost"
            size="icon-sm"
          >
            <Languages className="transition-transform duration-200 hover:rotate-12 hover:scale-110" />
            <span className="sr-only">Select language - Current: {currentLanguage.nativeName}</span>
          </Button>
        )}
      />
      <DropdownMenuContent align="end" className="w-48">
        {LANGUAGES.map((language) => (
          <DropdownMenuItem
            key={language.code}
            onClick={() => i18n.changeLanguage(language.code)}
            className={cn(
              'flex items-center justify-between cursor-pointer',
              i18n.language === language.code && 'bg-accent',
            )}
          >
            <span className="flex items-center gap-2">
              <span className="text-sm">{language.nativeName}</span>
              <span className="text-xs text-muted-foreground">({language.name})</span>
            </span>
            {i18n.language === language.code && <Check className="h-4 w-4 text-primary" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
