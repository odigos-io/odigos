import React from 'react';
import { useKeyDown } from '@odigos/ui-utils';
import { useSourceCRUD, useSourceFormData } from '@/hooks';
import { ChooseSourcesBody } from '../choose-sources-body';
import { Modal, NavigationButtons } from '@odigos/ui-components';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const AddSourceModal: React.FC<Props> = ({ isOpen, onClose }) => {
  useKeyDown({ key: 'Enter', active: isOpen }, () => handleSubmit());

  const menuState = useSourceFormData();
  const { persistSources, loading } = useSourceCRUD({ onSuccess: onClose });

  const handleSubmit = async () => {
    const { getApiSourcesPayload, getApiFutureAppsPayload } = menuState;

    await persistSources(getApiSourcesPayload(), getApiFutureAppsPayload());
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
              variant: 'primary',
              onClick: handleSubmit,
              disabled: loading,
            },
          ]}
        />
      }
    >
      <ChooseSourcesBody componentType='FAST' isModal {...menuState} />
    </Modal>
  );
};
