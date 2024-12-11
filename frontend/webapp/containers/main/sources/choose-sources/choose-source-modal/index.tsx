import React from 'react';
import { ChooseSourcesBody } from '../choose-sources-body';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { useKeyDown, useSourceCRUD, useSourceFormData } from '@/hooks';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const AddSourceModal: React.FC<Props> = ({ isOpen, onClose }) => {
  useKeyDown({ key: 'Enter', active: isOpen }, () => handleSubmit());

  const menuState = useSourceFormData();
  const { createSources } = useSourceCRUD({ onSuccess: onClose });

  const handleSubmit = async () => {
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
              onClick: handleSubmit,
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
