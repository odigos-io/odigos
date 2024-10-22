import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { Button, Text } from '@/reuseable-components';

const StyledAddDestinationButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
`;

interface ModalActionComponentProps {
  onClick: () => void;
}

export function AddDestinationButton({ onClick }: ModalActionComponentProps) {
  return (
    <StyledAddDestinationButton variant="secondary" onClick={onClick}>
      <Image src="/icons/common/plus.svg" alt="back" width={16} height={16} />
      <Text
        color={theme.colors.secondary}
        size={14}
        decoration={'underline'}
        family="secondary"
      >
        ADD DESTINATION
      </Text>
    </StyledAddDestinationButton>
  );
}
