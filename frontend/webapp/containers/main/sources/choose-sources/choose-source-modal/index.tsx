import React, { useState, useCallback } from 'react';
import { ChooseSourcesBody } from '../choose-sources-body';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { K8sActualSource, PersistNamespaceItemInput } from '@/types';
import { useActualSources, useConnectSourcesMenuState } from '@/hooks';

interface AddSourceModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddSourceModal: React.FC<AddSourceModalProps> = ({ isOpen, onClose }) => {
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({ sourcesList });
  const { createSourcesForNamespace, persistNamespaceItems } = useActualSources();

  const handleNextClick = useCallback(async () => {
    try {
      const namespaceItems: PersistNamespaceItemInput[] = Object.entries(stateMenu.futureAppsCheckbox).map(([namespaceName, futureSelected]) => ({
        name: namespaceName,
        futureSelected,
      }));

      await persistNamespaceItems(namespaceItems);

      await Promise.all(
        Object.entries(stateMenu.selectedItems).map(async ([namespaceName, sources]) => {
          const formattedSources = sources.map((source) => ({
            kind: source.kind,
            name: source.name,
            selected: true,
          }));
          await createSourcesForNamespace(namespaceName, formattedSources);
        })
      );

      onClose();
    } catch (error) {
      console.error('Error during handleNextClick:', error);
    }
  }, [stateMenu, onClose, createSourcesForNamespace, persistNamespaceItems]);

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
