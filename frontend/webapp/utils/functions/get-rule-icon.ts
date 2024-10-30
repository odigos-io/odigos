import type { InstrumentationRuleType } from '@/types';

const ICON_PATH = '/icons/rules/';

export const getRuleIcon = (type: InstrumentationRuleType) => {
  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `${ICON_PATH}${typeLowerCased}.svg`;
};
