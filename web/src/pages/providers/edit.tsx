import { useNavigate, useParams } from 'react-router-dom';
import { useProviders } from '@/hooks/queries';
import { ProviderEditFlow } from './components/provider-edit-flow';
import { useTranslation } from 'react-i18next';
import { PageHeader } from '@/components/layout/page-header';
import { Server, Loader2 } from 'lucide-react';

export function ProviderEditPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { data: providers, isLoading } = useProviders();

  const provider = providers?.find((p) => p.id + '' === id + '');

  if (isLoading) {
    return (
      <div className="flex flex-col h-full bg-background">
        <PageHeader
          icon={Server}
          iconClassName="text-blue-500"
          title={t('common.loading')}
        />
        <div className="flex items-center justify-center flex-1">
          <Loader2 className="h-8 w-8 animate-spin text-accent" />
        </div>
      </div>
    );
  }

  if (!provider) {
    return (
      <div className="flex flex-col h-full bg-background">
        <PageHeader
          icon={Server}
          iconClassName="text-blue-500"
          title={t('providers.notFound')}
        />
        <div className="flex items-center justify-center flex-1">
          <div className="text-muted-foreground">{t('providers.notFound')}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-background">
      <PageHeader
        icon={Server}
        iconClassName="text-blue-500"
        title={provider.name}
        description={t('providers.edit')}
      />
      <div className="flex-1 overflow-auto">
        <ProviderEditFlow provider={provider} onClose={() => navigate('/providers')} />
      </div>
    </div>
  );
}
