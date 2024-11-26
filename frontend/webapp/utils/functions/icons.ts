import { type ActionsType, type InstrumentationRuleType, type NotificationType, OVERVIEW_ENTITY_TYPES } from '@/types';

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

export const getEntityIcon = (type?: OVERVIEW_ENTITY_TYPES) => {
  if (!type) return BRAND_ICON;

  return `/icons/overview/${type}s.svg`;
};

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `/icons/rules/${typeLowerCased}.svg`;
};

export const getActionIcon = (type?: ActionsType | 'sampler' | 'attributes') => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');
  const isAttributes = typeLowerCased === 'attributes';

  const iconName = isSampler ? 'sampler' : isAttributes ? 'piimasking' : typeLowerCased;

  return `/icons/actions/${iconName}.svg`;
};
