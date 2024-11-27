import React from 'react';
import { useModalStore } from '@/store';
import { ActionModal } from '../../actions';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { DestinationModal } from '../../destinations';
import { RuleModal } from '../../instrumentation-rules';
import { AddSourceModal } from '../../sources/choose-sources/choose-source-modal';

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
