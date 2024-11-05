import type { ActionsType, InstrumentationRuleType } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getStatusIcon = (status?: 'success' | 'error' | 'info') => {
  if (!status) return BRAND_ICON;

  switch (status) {
    case 'success':
      return '/icons/notification/success-icon.svg';

    case 'error':
      return '/icons/notification/error-icon2.svg';

    case 'info':
      return '/icons/common/info.svg';

    default:
      return BRAND_ICON;
  }
};

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `/icons/rules/${typeLowerCased}.svg`;
};

export const getActionIcon = (type?: ActionsType | 'sampler') => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');

  return `/icons/actions/${isSampler ? 'sampler' : typeLowerCased}.svg`;
};
