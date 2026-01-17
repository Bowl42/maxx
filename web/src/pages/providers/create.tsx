import { useNavigate } from 'react-router-dom';
import { ProviderCreateFlow } from './components/provider-create-flow';

export function ProviderCreatePage() {
  const navigate = useNavigate();

  return <ProviderCreateFlow onClose={() => navigate('/providers')} />;
}
