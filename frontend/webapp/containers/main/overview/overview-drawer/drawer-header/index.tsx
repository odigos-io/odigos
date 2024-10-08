// DrawerHeader.tsx
import React, { useEffect, useState, forwardRef } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { Button, Input, Text } from '@/reuseable-components';

const HeaderContainer = styled.section`
  display: flex;
  height: 76px;
  padding: 0px 32px;
  justify-content: space-between;
  flex-shrink: 0;
  align-self: stretch;
  border-bottom: 1px solid rgba(249, 249, 249, 0.24);
`;

const SectionItemsWrapper = styled.div<{ gap?: number }>`
  display: flex;
  align-items: center;
  gap: ${({ gap }) => gap || 16}px;
`;

const InputWrapper = styled(SectionItemsWrapper)`
  width: 75%;
`;

const Title = styled(Text)`
  font-size: 18px;
  line-height: 26px;
`;

const DrawerItemImageWrapper = styled.div`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: linear-gradient(
    180deg,
    rgba(249, 249, 249, 0.06) 0%,
    rgba(249, 249, 249, 0.02) 100%
  );
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

interface DrawerHeaderProps {
  title: string;
  imageUri: string;
  isEditing: boolean;
  setIsEditing: (isEditing: boolean) => void;
}

const DrawerHeader = forwardRef<HTMLInputElement, DrawerHeaderProps>(
  ({ title, imageUri, isEditing, setIsEditing }, ref) => {
    const [inputValue, setInputValue] = useState(title);

    useEffect(() => {
      setInputValue(title);
    }, [title]);

    return (
      <HeaderContainer>
        <SectionItemsWrapper>
          <DrawerItemImageWrapper>
            <Image src={imageUri} alt="Drawer Item" width={16} height={16} />
          </DrawerItemImageWrapper>
          {!isEditing && <Title>{title}</Title>}
        </SectionItemsWrapper>
        {isEditing && (
          <InputWrapper>
            <Input
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              autoFocus
              ref={ref}
            />
          </InputWrapper>
        )}
        <SectionItemsWrapper gap={8}>
          {!isEditing && (
            <EditButton variant="tertiary" onClick={() => setIsEditing(true)}>
              <Image
                src="/icons/common/edit.svg"
                alt="Edit"
                width={16}
                height={16}
              />
              <ButtonText>Edit</ButtonText>
            </EditButton>
          )}
          <CloseButton variant="secondary" onClick={() => setIsEditing(false)}>
            <Image
              src="/icons/common/x.svg"
              alt="Close"
              width={11}
              height={10}
            />
          </CloseButton>
        </SectionItemsWrapper>
      </HeaderContainer>
    );
  }
);

export default DrawerHeader;
