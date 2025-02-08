import React from 'react';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useModalStore } from '@odigos/ui-containers';
import { AddSourceModal, DestinationModal } from '@/containers';

const AllModals = () => {
  const { currentModal, setCurrentModal } = useModalStore();
  const handleClose = () => setCurrentModal('');

  switch (currentModal) {
    case ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    case ENTITY_TYPES.DESTINATION:
      return <DestinationModal isOpen onClose={handleClose} />;

    default:
      return null;
  }
};

export default AllModals;
