import type { ActionsType } from '@/types';

const ICON_PATH = '/icons/actions/';

export const getActionIcon = (type?: ActionsType | 'sampler') => {
  if (!type) return '/brand/odigos-icon.svg';

  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');

  return `${ICON_PATH}${isSampler ? 'sampler' : typeLowerCased}.svg`;
};
