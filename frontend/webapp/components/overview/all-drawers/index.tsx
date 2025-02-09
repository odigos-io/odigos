import React from 'react';
import { SourceDrawer } from '@/containers';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useDrawerStore } from '@odigos/ui-containers';

const AllDrawers = () => {
  const { drawerType } = useDrawerStore();

  switch (drawerType) {
    case ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    default:
      return null;
  }
};

export default AllDrawers;
