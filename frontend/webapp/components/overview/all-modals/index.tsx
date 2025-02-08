import React from 'react';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useInstrumentationRuleCRUD } from '@/hooks';
import { ActionModal, AddSourceModal, DestinationModal } from '@/containers';
import { InstrumentationRuleModal, useModalStore } from '@odigos/ui-containers';

const AllModals = () => {
  const { currentModal, setCurrentModal } = useModalStore();
  const handleClose = () => setCurrentModal('');

  const { createInstrumentationRule } = useInstrumentationRuleCRUD();

  switch (currentModal) {
    case ENTITY_TYPES.INSTRUMENTATION_RULE:
      return <InstrumentationRuleModal isEnterprise={false} createInstrumentationRule={createInstrumentationRule} />;

    case ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.ACTION:
      return <ActionModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationModal isOpen onClose={handleClose} />;

    default:
      return null;
  }
};

export default AllModals;
