import React, { useState } from 'react';
import type { K8sActualSource } from '@/types';
import { ChooseSourcesBody } from '../choose-sources-body';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { useConnectSourcesMenuState, useSourceCRUD } from '@/hooks';

interface AddSourceModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddSourceModal: React.FC<AddSourceModalProps> = ({ isOpen, onClose }) => {
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({ sourcesList });
  const { createSources } = useSourceCRUD({ onSuccess: onClose });

  const handleNextClick = async () => {
    await createSources(stateMenu.selectedItems, stateMenu.futureAppsCheckbox);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      header={{ title: 'Add Source' }}
      actionComponent={
        <NavigationButtons
          buttons={[
            {
              label: 'DONE',
              onClick: handleNextClick,
              variant: 'primary',
            },
          ]}
        />
      }
    >
      <ChooseSourcesBody isModal stateMenu={stateMenu} sourcesList={sourcesList} stateHandlers={stateHandlers} setSourcesList={setSourcesList} />
    </Modal>
  );
};
