import React from 'react';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { DescribeDrawer } from './describe-drawer';
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

    case DRAWER_OTHER_TYPES.DESCRIBE_ODIGOS:
      return <DescribeDrawer />;

    default:
      return <></>;
  }
};

export default AllDrawers;
