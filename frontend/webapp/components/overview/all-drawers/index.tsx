import React from 'react';
import { CliDrawer } from './cli-drawer';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@/store';
import { ActionDrawer, DestinationDrawer, RuleDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const selected = useDrawerStore(({ selectedItem }) => selectedItem);

  if (!selected?.type) return null;

  switch (selected.type) {
    case OVERVIEW_ENTITY_TYPES.RULE:
      return <RuleDrawer />;

    case OVERVIEW_ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      return <ActionDrawer />;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    case DRAWER_OTHER_TYPES.ODIGOS_CLI:
      return <CliDrawer />;

    default:
      return <></>;
  }
};

export default AllDrawers;
