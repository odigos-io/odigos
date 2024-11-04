import React from 'react';
import { useModalStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { AddRuleModal } from '../../instrumentation-rules';
import { AddActionModal } from '../../actions';
import { AddDestinationModal } from '../../destinations/add-destination/add-destination-modal';
import { AddSourceModal } from '../../sources/choose-sources/choose-source-modal';

const AllModals = () => {
  const selected = useModalStore(({ currentModal }) => currentModal);
  const setSelected = useModalStore(({ setCurrentModal }) => setCurrentModal);

  if (!selected) return null;

  const handleClose = () => setSelected('');

  switch (selected) {
    case OVERVIEW_ENTITY_TYPES.RULE:
      return <AddRuleModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.SOURCE:
      return <AddSourceModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      return <AddActionModal isOpen onClose={handleClose} />;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      return <AddDestinationModal isOpen onClose={handleClose} />;

    default:
      return <></>;
  }
};

export default AllModals;
