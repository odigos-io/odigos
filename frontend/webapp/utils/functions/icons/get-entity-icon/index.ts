import { OVERVIEW_ENTITY_TYPES } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getEntityIcon = (type?: OVERVIEW_ENTITY_TYPES) => {
  if (!type) return BRAND_ICON;

  return `/icons/overview/${type}s.svg`;
};
