// DrawerHeader.tsx
import React, { useState } from 'react';
import styled from 'styled-components';
import { Input, Text } from '@/reuseable-components'; // Adjust the path if needed

const HeaderContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background-color: ${({ theme }) => theme.colors.translucent_bg};
  border-bottom: 1px solid rgba(249, 249, 249, 0.24);
`;

const Title = styled(Text)`
  font-size: 18px;
  font-weight: 600;
`;

const EditIcon = styled.div`
  cursor: pointer;
  /* Add styling for the edit icon if necessary */
`;

interface DrawerHeaderProps {
  title: string;
  onSave: (newTitle: string) => void;
  isEditing: boolean;
  setIsEditing: (isEditing: boolean) => void;
}

const DrawerHeader: React.FC<DrawerHeaderProps> = ({
  title,
  onSave,
  isEditing,
  setIsEditing,
}) => {
  const [inputValue, setInputValue] = useState(title);

  const handleSave = () => {
    onSave(inputValue);
    setIsEditing(false);
  };

  return (
    <HeaderContainer>
      {isEditing ? (
        <Input
          initialValue={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          autoFocus
        />
      ) : (
        <>
          <Title>{title}</Title>
          <EditIcon onClick={() => setIsEditing(true)}>
            {/* Replace with an actual icon if needed */}
            ✏️
          </EditIcon>
        </>
      )}
    </HeaderContainer>
  );
};

export default DrawerHeader;
