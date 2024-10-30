import type { ActionsType } from '@/types';

const ICON_PATH = '/icons/actions/';

export const getActionIcon = (type: ActionsType | 'sampler') => {
  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');

  return `${ICON_PATH}${isSampler ? 'sampler' : typeLowerCased}.svg`;
};
