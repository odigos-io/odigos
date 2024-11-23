import React from 'react';
import { ChooseSourcesBody } from '../choose-sources-body';
import { useSourceCRUD, useSourceFormData } from '@/hooks';
import { Modal, NavigationButtons } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const AddSourceModal: React.FC<Props> = ({ isOpen, onClose }) => {
  const menuState = useSourceFormData();
  const { createSources } = useSourceCRUD({ onSuccess: onClose });

  const handleNextClick = async () => {
    const { selectedSources, selectedFutureApps } = menuState;

    await createSources(selectedSources, selectedFutureApps);
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
      <ChooseSourcesBody componentType='FAST' isModal {...menuState} />
    </Modal>
  );
};
