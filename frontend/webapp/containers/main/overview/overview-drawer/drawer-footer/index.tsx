// DrawerFooter.tsx
import React from 'react';
import styled from 'styled-components';
import { Button } from '@/reuseable-components'; // Adjust the path if needed

const FooterContainer = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 16px;
  background-color: ${({ theme }) => theme.colors.translucent_bg};
  border-top: 1px solid rgba(249, 249, 249, 0.24);
`;

interface DrawerFooterProps {
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
}

const DrawerFooter: React.FC<DrawerFooterProps> = ({
  onSave,
  onCancel,
  onDelete,
}) => (
  <FooterContainer>
    <Button variant="primary" onClick={onSave}>
      Save
    </Button>
    <Button variant="secondary" onClick={onCancel}>
      Cancel
    </Button>
    <Button variant="tertiary" onClick={onDelete}>
      Delete
    </Button>
  </FooterContainer>
);

export default DrawerFooter;
