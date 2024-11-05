import type { ActionsType, InstrumentationRuleType } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getStatusIcon = (active?: boolean) => {
  const path = '/icons/notification/';

  return `${path}${active ? 'success-icon' : 'error-icon2'}.svg`;
};

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return BRAND_ICON;

  const path = '/icons/rules/';
  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `${path}${typeLowerCased}.svg`;
};

export const getActionIcon = (type?: ActionsType | 'sampler') => {
  if (!type) return BRAND_ICON;

  const path = '/icons/actions/';
  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');

  return `${path}${isSampler ? 'sampler' : typeLowerCased}.svg`;
};
