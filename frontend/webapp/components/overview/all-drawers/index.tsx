import React from 'react';
import { CliDrawer } from './cli-drawer';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@odigos/ui-containers';
import { ActionDrawer, DestinationDrawer, RuleDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const { drawerType } = useDrawerStore();

  switch (drawerType) {
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
      return null;
  }
};

export default AllDrawers;
