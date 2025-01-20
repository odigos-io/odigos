import { ActionsType } from '@/types';
import { AddClusterInfoIcon, DeleteAttributeIcon, PiiMaskingIcon, RenameAttributeIcon, SamplerIcon, SVG } from '@/assets';

export const getActionIcon = (type: ActionsType | 'sampler' | 'attributes') => {
  const typeLowerCased = type?.toLowerCase();
  const isSamplerCategory = typeLowerCased?.includes('sampler');
  const isAttributesCategory = typeLowerCased === 'attributes';

  if (isSamplerCategory) return SamplerIcon;
  if (isAttributesCategory) return PiiMaskingIcon;

  const LOGOS: Record<ActionsType, SVG> = {
    [ActionsType.ADD_CLUSTER_INFO]: AddClusterInfoIcon,
    [ActionsType.DELETE_ATTRIBUTES]: DeleteAttributeIcon,
    [ActionsType.PII_MASKING]: PiiMaskingIcon,
    [ActionsType.RENAME_ATTRIBUTES]: RenameAttributeIcon,
    [ActionsType.ERROR_SAMPLER]: SamplerIcon,
    [ActionsType.PROBABILISTIC_SAMPLER]: SamplerIcon,
    [ActionsType.LATENCY_SAMPLER]: SamplerIcon,
  };

  return LOGOS[type as ActionsType];
};
