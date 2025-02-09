import React from 'react';
import { AddSourceModal } from '@/containers';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useModalStore } from '@odigos/ui-containers';

const AllModals = () => {
  const { currentModal, setCurrentModal } = useModalStore();
  const handleClose = () => setCurrentModal('');

  switch (currentModal) {
    case ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    default:
      return null;
  }
};

export default AllModals;
