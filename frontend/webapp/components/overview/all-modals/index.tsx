import React from 'react';
import { useModalStore } from '@/store';
import { ENTITY_TYPES } from '@odigos/ui-components';
import { ActionModal, AddSourceModal, DestinationModal, RuleModal } from '@/containers';

const AllModals = () => {
  const selected = useModalStore(({ currentModal }) => currentModal);
  const setSelected = useModalStore(({ setCurrentModal }) => setCurrentModal);

  if (!selected) return null;

  const handleClose = () => setSelected('');

  switch (selected) {
    case ENTITY_TYPES.INSTRUMENTATION_RULE:
      return <RuleModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.ACTION:
      return <ActionModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationModal isOpen onClose={handleClose} />;

    default:
      return <></>;
  }
};

export default AllModals;
