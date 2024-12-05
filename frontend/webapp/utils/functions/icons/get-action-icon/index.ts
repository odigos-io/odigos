import { type ActionsType } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getActionIcon = (type?: ActionsType | 'sampler' | 'attributes') => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');
  const isAttributes = typeLowerCased === 'attributes';

  const iconName = isSampler ? 'sampler' : isAttributes ? 'piimasking' : typeLowerCased;

  return `/icons/actions/${iconName}.svg`;
};
