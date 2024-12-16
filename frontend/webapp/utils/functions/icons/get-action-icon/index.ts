import { ActionsType } from '@/types';
import { AddClusterInfoIcon } from '@/assets';

export const getActionIcon = (type?: ActionsType | 'sampler' | 'attributes') => {
  // if (!type) return '';

  const typeLowerCased = type?.toLowerCase();
  const isSamplerCategory = typeLowerCased?.includes('sampler');
  const isAttributesCategory = typeLowerCased === 'attributes';

  // if (isSamplerCategory) return SamplerIcon;
  // if (isAttributesCategory) return PiiMaskingIcon;

  switch (type) {
    case ActionsType.ADD_CLUSTER_INFO:
      return AddClusterInfoIcon;

    default:
      return undefined;
  }
};
