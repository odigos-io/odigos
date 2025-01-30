import React from 'react';
import { CliDrawer } from './cli-drawer';
import { Types } from '@odigos/ui-components';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@/store';
import { ActionDrawer, DestinationDrawer, RuleDrawer, SourceDrawer } from '@/containers';

const AllDrawers = () => {
  const selected = useDrawerStore(({ selectedItem }) => selectedItem);

  if (!selected?.type) return null;

  switch (selected.type) {
    case Types.ENTITY_TYPES.INSTRUMENTATION_RULE:
      return <RuleDrawer />;

    case Types.ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case Types.ENTITY_TYPES.ACTION:
      return <ActionDrawer />;

    case Types.ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    case DRAWER_OTHER_TYPES.ODIGOS_CLI:
      return <CliDrawer />;

    default:
      return <></>;
  }
};

export default AllDrawers;
