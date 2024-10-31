import type { InstrumentationRuleType } from '@/types';

const ICON_PATH = '/icons/rules/';

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return '/brand/odigos-icon.svg';

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `${ICON_PATH}${typeLowerCased}.svg`;
};
