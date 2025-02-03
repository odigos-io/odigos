import React from 'react';
import { CliDrawer } from './cli-drawer';
import { ENTITY_TYPES } from '@odigos/ui-components';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@/store';
import { ActionDrawer, DestinationDrawer, RuleDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const selected = useDrawerStore(({ selectedItem }) => selectedItem);

  if (!selected?.type) return null;

  switch (selected.type) {
    case ENTITY_TYPES.INSTRUMENTATION_RULE:
      return <RuleDrawer />;

    case ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case ENTITY_TYPES.ACTION:
      return <ActionDrawer />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    case DRAWER_OTHER_TYPES.ODIGOS_CLI:
      return <CliDrawer />;

    default:
      return <></>;
  }
};

export default AllDrawers;
