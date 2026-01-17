import { useTranslation } from 'react-i18next';
import { Activity } from 'lucide-react';
import { useStreamingRequests } from '@/hooks/use-streaming';
import { AnimatedNavItem } from './animated-nav-item';

/**
 * Requests navigation item with streaming badge and marquee animation
 */
export function RequestsNavItem() {
  const { total } = useStreamingRequests();
  const { t } = useTranslation();
  const color = 'var(--color-success)'; // emerald-500

  return (
    <AnimatedNavItem
      to="/requests"
      isActive={(pathname) => pathname === '/requests' || pathname.startsWith('/requests/')}
      tooltip={t('requests.title')}
      icon={<Activity />}
      label={t('requests.title')}
      streamingCount={total}
      color={color}
    />
  );
}
