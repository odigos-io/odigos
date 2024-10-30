import styled from 'styled-components';
import React, { useState, useCallback } from 'react';
import { useConnectSourcesMenuState } from '@/hooks';
import { ChooseSourcesBody } from '../choose-sources-body';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { K8sActualSource, PersistNamespaceItemInput } from '@/types';

const ChooseSourcesBodyWrapper = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
`;

interface AddSourceModalProps {
  isOpen: boolean;
  onClose: () => void;
  createSourcesForNamespace: (namespaceName: string, sources: { kind: string; name: string; selected: boolean }[]) => Promise<void>;
  persistNamespaceItems: (namespaceItems: PersistNamespaceItemInput[]) => Promise<void>;
}

export const AddSourceModal: React.FC<AddSourceModalProps> = ({ isOpen, onClose, createSourcesForNamespace, persistNamespaceItems }) => {
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({
    sourcesList,
  });

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
  }, [createSourcesForNamespace, persistNamespaceItems, stateMenu, onClose]);

  const ModalActionComponent = (
    <NavigationButtons
      buttons={[
        {
          label: 'DONE',
          onClick: handleNextClick,
          variant: 'primary',
        },
      ]}
    />
  );

  return (
    <Modal isOpen={isOpen} header={{ title: 'Add Source' }} actionComponent={ModalActionComponent} onClose={onClose}>
      <ChooseSourcesBodyWrapper>
        <ChooseSourcesBody isModal stateMenu={stateMenu} sourcesList={sourcesList} stateHandlers={stateHandlers} setSourcesList={setSourcesList} />
      </ChooseSourcesBodyWrapper>
    </Modal>
  );
};
