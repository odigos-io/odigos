import React, { useEffect, useState, forwardRef, useImperativeHandle } from 'react';
import styled, { useTheme } from 'styled-components';
import { EditIcon, SVG, TrashIcon, XIcon } from '@/assets';
import { Button, IconWrapped, Input, Text, Tooltip } from '@/reuseable-components';

const HeaderContainer = styled.section`
  display: flex;
  height: 76px;
  padding: 0px 32px;
  justify-content: space-between;
  flex-shrink: 0;
  align-self: stretch;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
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
  max-width: 270px;
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
  onDelete?: () => void;
  deleteLabel?: string;
}

const DrawerHeader = forwardRef<DrawerHeaderRef, DrawerHeaderProps>(({ title, titleTooltip, icon, iconSrc, isEdit, onEdit, onClose, onDelete, deleteLabel = 'Delete' }, ref) => {
  const theme = useTheme();
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

      <SectionItemsWrapper $gap={2}>
        {!!onEdit && !isEdit && (
          <EditButton data-id='drawer-edit' variant='tertiary' onClick={onEdit}>
            <EditIcon />
            <ButtonText>Edit</ButtonText>
          </EditButton>
        )}

        {!!onDelete && !isEdit && (
          <EditButton data-id='drawer-delete' variant='tertiary' onClick={onDelete}>
            <TrashIcon />
            <Text color={theme.text.error} size={14} family='secondary'>
              {deleteLabel}
            </Text>
          </EditButton>
        )}

        <CloseButton data-id='drawer-close' variant='secondary' onClick={onClose}>
          <XIcon size={12} />
        </CloseButton>
      </SectionItemsWrapper>
    </HeaderContainer>
  );
});

DrawerHeader.displayName = 'DrawerHeader';

export default DrawerHeader;
