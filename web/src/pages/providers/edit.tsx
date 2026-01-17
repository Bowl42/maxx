import { useNavigate, useParams } from 'react-router-dom';
import { useProviders } from '@/hooks/queries';
import { ProviderEditFlow } from './components/provider-edit-flow';
import { useTranslation } from 'react-i18next';

export function ProviderEditPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { data: providers, isLoading } = useProviders();

  const provider = providers?.find((p) => p.id + '' === id + '');

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-text-muted">{t('common.loading')}</div>
      </div>
    );
  }

  if (!provider) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-text-muted">{t('providers.notFound')}</div>
      </div>
    );
  }

  return <ProviderEditFlow provider={provider} onClose={() => navigate('/providers')} />;
}
