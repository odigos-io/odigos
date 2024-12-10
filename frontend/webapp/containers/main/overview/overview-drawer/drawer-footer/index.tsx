import React from 'react';
import Image from 'next/image';
import { slide } from '@/styles';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { useTransition } from '@/hooks';
import { Button, Text } from '@/reuseable-components';

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
  width: 140px;
  font-size: 14px;
`;

const DrawerFooter: React.FC<Props> = ({ isOpen, onSave, saveLabel = 'Save', onCancel, cancelLabel = 'Cancel', onDelete, deleteLabel = 'Delete' }) => {
  const Transition = useTransition({
    container: FooterContainer,
    animateIn: slide.in['bottom'],
    animateOut: slide.out['bottom'],
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
          <Image src='/icons/common/trash.svg' alt='Delete' width={16} height={16} />
          <Text color={theme.text.error} size={14} family='secondary'>
            {deleteLabel}
          </Text>
        </FooterButton>
      </AlignRight>
    </Transition>
  );
};

export default DrawerFooter;
