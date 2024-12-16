import { type InstrumentationRuleType } from '@/types';

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return '';

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `/icons/rules/${typeLowerCased}.svg`;
};
