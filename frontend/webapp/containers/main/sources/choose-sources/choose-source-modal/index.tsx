import React from 'react';
import { type IAppState } from '@/store';
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
    const { getApiPaylod, selectedFutureApps } = menuState;

    // Type of "getApiPaylod()" is actually:
    // { [namespace: string]: Pick<K8sActualSource, 'name' | 'kind' | 'selected' | 'numberOfInstances'>[] };
    //
    // But we will force it as type:
    // { [namespace: string]: K8sActualSource[] };
    //
    // This forced type is to satisfy TypeScript,
    // while knowing that this doesn't break the onboarding flow in any-way...

    await persistSources(getApiPaylod() as IAppState['configuredSources'], selectedFutureApps);
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
