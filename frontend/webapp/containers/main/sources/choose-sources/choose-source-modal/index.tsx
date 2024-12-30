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
  const { persistSources } = useSourceCRUD({ onSuccess: onClose });

  const handleSubmit = async () => {
    const { availableSources, selectedSources, selectedFutureApps } = menuState;
    const payload: typeof availableSources = {};

    Object.entries(availableSources).forEach(([namespace, sources]) => {
      payload[namespace] = sources.map((source) => ({
        ...source,
        selected: !!selectedSources[namespace].find(({ kind, name }) => kind === source.kind && name === source.name),
      }));
    });

    await persistSources(payload, selectedFutureApps);
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
