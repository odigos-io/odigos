import { OVERVIEW_ENTITY_TYPES } from '@/types';

export const getEntityIcon = (type?: OVERVIEW_ENTITY_TYPES) => {
  if (!type) return '';

  return `/icons/overview/${type}s.svg`;
};
