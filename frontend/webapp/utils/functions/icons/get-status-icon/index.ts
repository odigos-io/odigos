import { NOTIFICATION_TYPE } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getStatusIcon = (status?: NOTIFICATION_TYPE) => {
  if (!status) return BRAND_ICON;

  switch (status) {
    case NOTIFICATION_TYPE.SUCCESS:
      return '/icons/notification/success-icon.svg';
    case NOTIFICATION_TYPE.ERROR:
      return '/icons/notification/error-icon2.svg';
    case NOTIFICATION_TYPE.WARNING:
      return '/icons/notification/warning-icon2.svg';
    case NOTIFICATION_TYPE.INFO:
      return '/icons/common/info.svg';
    default:
      return BRAND_ICON;
  }
};
