import React from 'react';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { ActionDrawer, DestinationDrawer, SourceDrawer } from '@/containers';
import { useDescribeOdigos, useInstrumentationRuleCRUD, useTokenCRUD } from '@/hooks';
import { CliDrawer, DRAWER_OTHER_TYPES, InstrumentationRuleDrawer, useDrawerStore } from '@odigos/ui-containers';

const AllDrawers = () => {
  const { drawerType } = useDrawerStore();
  const { data: describe } = useDescribeOdigos();
  const { tokens, updateToken } = useTokenCRUD();
  const { instrumentationRules, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  switch (drawerType) {
    case ENTITY_TYPES.INSTRUMENTATION_RULE:
      return <InstrumentationRuleDrawer instrumentationRules={instrumentationRules} updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />;

    case ENTITY_TYPES.SOURCE:
      return <SourceDrawer />;

    case ENTITY_TYPES.ACTION:
      return <ActionDrawer />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationDrawer />;

    case DRAWER_OTHER_TYPES.ODIGOS_CLI:
      return <CliDrawer tokens={tokens} saveToken={updateToken} describe={describe} />;

    default:
      return null;
  }
};

export default AllDrawers;
