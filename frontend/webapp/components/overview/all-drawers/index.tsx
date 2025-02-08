import React from 'react';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useDrawerStore } from '@odigos/ui-containers';
import { DestinationDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const { drawerType } = useDrawerStore();

  switch (drawerType) {
    case ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    default:
      return null;
  }
};

export default AllDrawers;
