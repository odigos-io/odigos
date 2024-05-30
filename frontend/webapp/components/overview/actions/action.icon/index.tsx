import React from 'react';
import { ActionsType } from '@/types';
import {
  AddClusterInfoIcon,
  DeleteAttributeIcon,
  RenameAttributeIcon,
} from '@keyval-dev/design-system';

export function ActionIcon({ type, ...props }) {
  switch (type) {
    case ActionsType.ADD_CLUSTER_INFO:
      return <AddClusterInfoIcon {...props} />;
    case ActionsType.RENAME_ATTRIBUTES:
      return <RenameAttributeIcon {...props} />;
    case ActionsType.DELETE_ATTRIBUTES:
      return <DeleteAttributeIcon {...props} />;
    default:
      return null;
  }
}
