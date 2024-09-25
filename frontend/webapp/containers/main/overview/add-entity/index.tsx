import React, { useState, useRef } from 'react';
import Image from 'next/image';
import styled, { css } from 'styled-components';
import {
  DropdownOption,
  K8sActualSource,
  PersistNamespaceItemInput,
} from '@/types';
import { Button, Modal, NavigationButtons, Text } from '@/reuseable-components';
import { ChooseSourcesBody } from '../../sources/choose-sources/choose-sources-body';
import {
  useActualSources,
  useOnClickOutside,
  useConnectSourcesMenuState,
} from '@/hooks';

interface AddEntityButtonDropdownProps {
  options?: DropdownOption[];
  placeholder?: string;
}

const Container = styled.div`
  position: relative;
  display: inline-block;
`;

const StyledButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 100px;
`;

const DropdownListContainer = styled.div`
  position: absolute;
  right: 0px;
  top: 48px;
  border-radius: 24px;
  width: 131px;
  overflow-y: auto;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  z-index: 9999;
  padding: 12px;
`;

const DropdownItem = styled.div<{ isSelected: boolean }>`
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 24px;
  gap: 8px;
  display: flex;

  align-items: center;
  &:hover {
    background: ${({ theme }) => theme.colors.white_opacity['008']};
  }
  ${({ isSelected }) =>
    isSelected &&
    css`
      background: rgba(68, 74, 217, 0.24);
    `}
`;

const ButtonText = styled(Text)`
  color: ${({ theme }) => theme.text.primary};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 600;
`;

const ChooseSourcesBodyWrapper = styled.div`
  width: 1080px;
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
`;

const OPTIONS = [
  {
    id: 'sources',
    value: 'Source',
  },
  {
    id: 'actions',
    value: 'Action',
  },
  {
    id: 'destinations',
    value: 'Destination',
  },
];

function ModalActionComponent({ onNext }: { onNext: () => void }) {
  return (
    <NavigationButtons
      buttons={[
        {
          label: 'DONE',
          onClick: onNext,
          variant: 'primary',
        },
      ]}
    />
  );
}

const AddEntityButtonDropdown: React.FC<AddEntityButtonDropdownProps> = ({
  options = OPTIONS,
  placeholder = 'ADD...',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [currentModal, setCurrentModal] = useState<string>('');
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

  const { createSourcesForNamespace, persistNamespaceItems } =
    useActualSources();
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({
    sourcesList,
  });

  useOnClickOutside(dropdownRef, () => setIsOpen(false));

  const handleToggle = () => {
    setIsOpen((prev) => !prev);
  };

  const handleSelect = (option: DropdownOption) => {
    setCurrentModal(option.id);
    setIsOpen(false);
  };

  async function onNextClick() {
    console.log('object');
    try {
      // Persist the selected namespaces
      const namespaceItems: PersistNamespaceItemInput[] = Object.entries(
        stateMenu.futureAppsCheckbox
      ).map(([namespaceName, futureSelected]) => ({
        name: namespaceName,
        futureSelected,
      }));

      await persistNamespaceItems(namespaceItems);

      // Create sources for each namespace
      for (const namespaceName in stateMenu.selectedItems) {
        const sources = stateMenu.selectedItems[namespaceName].map(
          (source) => ({
            kind: source.kind,
            name: source.name,
            selected: true,
          })
        );
        await createSourcesForNamespace(namespaceName, sources);
      }
      setCurrentModal('');
    } catch (error) {
      console.error('Error during onNextClick:', error);
    }
  }

  return (
    <Container ref={dropdownRef}>
      <StyledButton onClick={handleToggle}>
        <Image
          src="/icons/common/plus-black.svg"
          width={16}
          height={16}
          alt="Add"
        />
        <ButtonText size={14}>{placeholder}</ButtonText>
      </StyledButton>
      {isOpen && (
        <DropdownListContainer>
          {options.map((option) => (
            <DropdownItem
              key={option.id}
              isSelected={false}
              onClick={() => handleSelect(option)}
            >
              <Image
                src={`/icons/overview/${option.id}.svg`}
                width={16}
                height={16}
                alt="Add"
              />
              <Text size={14}>{option.value}</Text>
            </DropdownItem>
          ))}
        </DropdownListContainer>
      )}
      <Modal
        isOpen={currentModal === 'sources'}
        header={{ title: `ADD ${currentModal.toUpperCase()}` }}
        actionComponent={<ModalActionComponent onNext={onNextClick} />}
        onClose={() => setCurrentModal('')}
      >
        <ChooseSourcesBodyWrapper>
          <ChooseSourcesBody
            isModal
            stateMenu={stateMenu}
            sourcesList={sourcesList}
            stateHandlers={stateHandlers}
            setSourcesList={setSourcesList}
          />
        </ChooseSourcesBodyWrapper>
      </Modal>
    </Container>
  );
};

export { AddEntityButtonDropdown };
