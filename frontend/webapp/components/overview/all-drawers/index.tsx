import React from 'react';
import { useDrawerStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { ActionDrawer, DestinationDrawer, RuleDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const selected = useDrawerStore(({ selectedItem }) => selectedItem);

  if (!selected?.item) return null;

  switch (selected.type) {
    case OVERVIEW_ENTITY_TYPES.RULE:
      return <RuleDrawer />;

    case OVERVIEW_ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      return <ActionDrawer />;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    default:
      return <></>;
  }
};

export default AllDrawers;
