import React from 'react';
import { useModalStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { ActionModal, AddSourceModal, DestinationModal, RuleModal } from '@/containers';

const AllModals = () => {
  const selected = useModalStore(({ currentModal }) => currentModal);
  const setSelected = useModalStore(({ setCurrentModal }) => setCurrentModal);

  if (!selected) return null;

  const handleClose = () => setSelected('');

  switch (selected) {
    case OVERVIEW_ENTITY_TYPES.RULE:
      return <RuleModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      return <ActionModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      return <DestinationModal isOpen onClose={handleClose} />;

    default:
      return <></>;
  }
};

export default AllModals;
