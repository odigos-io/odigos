import { type NotificationType } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getStatusIcon = (status?: NotificationType) => {
  if (!status) return BRAND_ICON;

  switch (status) {
    case 'success':
      return '/icons/notification/success-icon.svg';
    case 'error':
      return '/icons/notification/error-icon2.svg';
    case 'warning':
      return '/icons/notification/warning-icon2.svg';
    case 'info':
      return '/icons/common/info.svg';
    default:
      return BRAND_ICON;
  }
};
