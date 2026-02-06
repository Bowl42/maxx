import { Zap } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface CLIProxyAPISwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
}

export function CLIProxyAPISwitch({ checked, onChange, disabled }: CLIProxyAPISwitchProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between p-3 bg-muted rounded-lg border border-border">
      <div className="flex items-center gap-2">
        <Zap size={16} className="text-amber-500" />
        <span className="text-sm font-medium">
          {t('common.useCLIProxyAPI', 'Use CLIProxyAPI')}
        </span>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
          disabled ? 'opacity-50 cursor-not-allowed' : ''
        } ${checked ? 'bg-primary' : 'bg-secondary'}`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  );
}
