import { useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useTransport } from '@/lib/transport';

interface LoginPageProps {
  onSuccess: (token: string) => void;
}

export function LoginPage({ onSuccess }: LoginPageProps) {
  const { t } = useTranslation();
  const { transport } = useTransport();
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const result = await transport.verifyPassword(password);
      if (result.success && result.token) {
        onSuccess(result.token);
      } else {
        setError(result.error || t('login.invalidPassword'));
      }
    } catch {
      setError(t('login.verifyFailed'));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 p-6">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-bold">{t('login.title')}</h1>
          <p className="text-muted-foreground text-sm">
            {t('login.description')}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Input
              type="password"
              placeholder={t('login.passwordPlaceholder')}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoFocus
              disabled={isLoading}
            />
            {error && (
              <p className="text-destructive text-sm">{error}</p>
            )}
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isLoading || !password}
          >
            {isLoading ? t('login.verifying') : t('login.submit')}
          </Button>
        </form>
      </div>
    </div>
  );
}
