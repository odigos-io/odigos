import React from 'react';
import { useTransition } from '@/hooks';
import styled, { useTheme } from 'styled-components';
import { Button, Text } from '@/reuseable-components';
import { Theme, TrashIcon } from '@odigos/ui-components';

interface Props {
  isOpen: boolean;
  onSave: () => void;
  saveLabel?: string;
  onCancel: () => void;
  cancelLabel?: string;
  onDelete: () => void;
  deleteLabel?: string;
}

const FooterContainer = styled.div<{ $isVisible: boolean }>`
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 24px 18px 24px 32px;
  border-top: 1px solid ${({ theme }) => theme.colors.border};
  background-color: ${({ theme }) => theme.colors.translucent_bg};
  transform: translateY(100%);
`;

const AlignRight = styled.div`
  margin-left: auto;
`;

const FooterButton = styled(Button)`
  min-width: 140px;
  font-size: 14px;
`;

const DrawerFooter: React.FC<Props> = ({ isOpen, onSave, saveLabel = 'Save', onCancel, cancelLabel = 'Cancel', onDelete, deleteLabel = 'Delete' }) => {
  const theme = useTheme();
  const Transition = useTransition({
    container: FooterContainer,
    animateIn: Theme.slide.in['bottom'],
    animateOut: Theme.slide.out['bottom'],
  });

  return (
    <Transition enter={isOpen}>
      <FooterButton data-id='drawer-save' variant='primary' onClick={onSave}>
        {saveLabel}
      </FooterButton>
      <FooterButton data-id='drawer-cancel' variant='secondary' onClick={onCancel}>
        {cancelLabel}
      </FooterButton>

      <AlignRight>
        <FooterButton data-id='drawer-delete' variant='tertiary' onClick={onDelete}>
          <TrashIcon />
          <Text size={14} color={theme.text.error} family='secondary' decoration='underline'>
            {deleteLabel}
          </Text>
        </FooterButton>
      </AlignRight>
    </Transition>
  );
};

export default DrawerFooter;
