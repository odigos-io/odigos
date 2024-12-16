import React, { useEffect, useState, forwardRef, useImperativeHandle } from 'react';
import Image from 'next/image';
import { SVG } from '@/assets';
import styled from 'styled-components';
import { Button, IconWrapped, Input, Text, Tooltip } from '@/reuseable-components';

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
  icon?: SVG;
  iconSrc?: string;
  isEdit?: boolean;
  onEdit?: () => void;
  onClose: () => void;
}

const DrawerHeader = forwardRef<DrawerHeaderRef, DrawerHeaderProps>(({ title, titleTooltip, icon, iconSrc, isEdit, onEdit, onClose }, ref) => {
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
        {(!!icon || !!iconSrc) && <IconWrapped icon={icon} src={iconSrc} alt='Drawer Item' />}

        {!isEdit && (
          <Tooltip text={titleTooltip} withIcon>
            <Title>{title}</Title>
          </Tooltip>
        )}
      </SectionItemsWrapper>

      {/* "titleTooltip" is currently used only by sources, if we add tooltip to other entities we will have to define a "hideTitleInput" prop */}
      {isEdit && !titleTooltip && (
        <InputWrapper>
          <Input data-id='title' autoFocus value={inputValue} onChange={(e) => setInputValue(e.target.value)} />
        </InputWrapper>
      )}

      <SectionItemsWrapper $gap={8}>
        {!isEdit && !!onEdit && (
          <EditButton data-id='drawer-edit' variant='tertiary' onClick={onEdit}>
            <Image src='/icons/common/edit.svg' alt='Edit' width={16} height={16} />
            <ButtonText>Edit</ButtonText>
          </EditButton>
        )}

        <CloseButton data-id='drawer-close' variant='secondary' onClick={onClose}>
          <Image src='/icons/common/x.svg' alt='Close' width={12} height={12} />
        </CloseButton>
      </SectionItemsWrapper>
    </HeaderContainer>
  );
});

DrawerHeader.displayName = 'DrawerHeader';

export default DrawerHeader;
