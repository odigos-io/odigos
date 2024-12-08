import React, { useEffect, useState, forwardRef, useImperativeHandle } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Button, Input, Text, Tooltip } from '@/reuseable-components';

const HeaderContainer = styled.section`
  display: flex;
  height: 76px;
  padding: 0px 32px;
  justify-content: space-between;
  flex-shrink: 0;
  align-self: stretch;
  border-bottom: 1px solid rgba(249, 249, 249, 0.24);
`;

const SectionItemsWrapper = styled.div<{ $gap?: number }>`
  display: flex;
  align-items: center;
  gap: ${({ $gap }) => $gap || 16}px;
`;

const InputWrapper = styled(SectionItemsWrapper)`
  width: 75%;
`;

const Title = styled(Text)`
  font-size: 18px;
  line-height: 26px;
  max-width: 400px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`;

const DrawerItemImageWrapper = styled.div`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(249, 249, 249, 0.06) 0%, rgba(249, 249, 249, 0.02) 100%);
`;

const EditButton = styled(Button)`
  gap: 8px;
`;

const CloseButton = styled(Button)``;

const ButtonText = styled(Text)`
  font-size: 14px;
  font-weight: 600;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-decoration-line: underline;
  text-transform: uppercase;
  width: fit-content;
`;

export interface DrawerHeaderRef {
  getTitle: () => string;
  clearTitle: () => void;
}

interface DrawerHeaderProps {
  title: string;
  titleTooltip?: string;
  imageUri: string;
  isEdit: boolean;
  onEdit: () => void;
  onClose: () => void;
}

const DrawerHeader = forwardRef<DrawerHeaderRef, DrawerHeaderProps>(({ title, titleTooltip, imageUri, isEdit, onEdit, onClose }, ref) => {
  const [inputValue, setInputValue] = useState(title);

  useEffect(() => {
    setInputValue(title);
  }, [title]);

  useImperativeHandle(ref, () => ({
    getTitle: () => inputValue,
    clearTitle: () => setInputValue(title),
  }));

  return (
    <HeaderContainer>
      <SectionItemsWrapper>
        <DrawerItemImageWrapper>
          <Image src={imageUri} alt='Drawer Item' width={16} height={16} />
        </DrawerItemImageWrapper>
        {!isEdit && (
          <Tooltip text={titleTooltip} withIcon>
            <Title>{title}</Title>
          </Tooltip>
        )}
      </SectionItemsWrapper>

      {/* "titleTooltip" is currently used only by sources, if we add tooltip to other entities we will have to define a "hideTitleInput" prop */}
      {isEdit && !titleTooltip && (
        <InputWrapper>
          <Input id='title' autoFocus value={inputValue} onChange={(e) => setInputValue(e.target.value)} />
        </InputWrapper>
      )}

      <SectionItemsWrapper $gap={8}>
        {!isEdit && (
          <EditButton id='drawer-edit' variant='tertiary' onClick={onEdit}>
            <Image src='/icons/common/edit.svg' alt='Edit' width={16} height={16} />
            <ButtonText>Edit</ButtonText>
          </EditButton>
        )}
        <CloseButton id='drawer-close' variant='secondary' onClick={onClose}>
          <Image src='/icons/common/x.svg' alt='Close' width={12} height={12} />
        </CloseButton>
      </SectionItemsWrapper>
    </HeaderContainer>
  );
});

DrawerHeader.displayName = 'DrawerHeader';

export default DrawerHeader;
