import { type InstrumentationRuleType } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `/icons/rules/${typeLowerCased}.svg`;
};
