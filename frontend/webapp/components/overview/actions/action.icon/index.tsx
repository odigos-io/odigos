import React from 'react';
import { ActionsType } from '@/types';
import {
  AddClusterInfoIcon,
  DeleteAttributeIcon,
  ErrorSamplerIcon,
  RenameAttributeIcon,
  ProbabilisticSamplerIcon,
  LatencySamplerIcon,
  PiiMaskingIcon,
} from '@keyval-dev/design-system';

export function ActionIcon({ type, ...props }) {
  switch (type) {
    case ActionsType.ADD_CLUSTER_INFO:
      return <AddClusterInfoIcon {...props} />;
    case ActionsType.RENAME_ATTRIBUTES:
      return <RenameAttributeIcon {...props} />;
    case ActionsType.DELETE_ATTRIBUTES:
      return <DeleteAttributeIcon {...props} />;
    case ActionsType.ERROR_SAMPLER:
      return <ErrorSamplerIcon {...props} />;
    case ActionsType.PROBABILISTIC_SAMPLER:
      return <ProbabilisticSamplerIcon {...props} />;
    case ActionsType.LATENCY_SAMPLER:
      return <LatencySamplerIcon {...props} />;
    case ActionsType.PII_MASKING:
      return <PiiMaskingIcon {...props} />;
    default:
      return null;
  }
}
