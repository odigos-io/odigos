import React from 'react';
import { Text } from '../text';
import styled from 'styled-components';

const ErrorWrapper = styled.div`
  padding: 4px 0 0 0;
`;

const ErrorMessage = styled(Text)`
  font-size: 12px;
  color: ${({ theme }) => theme.text.error};
`;

export const FieldError: React.FC<React.PropsWithChildren> = ({ children }) => {
  return (
    <ErrorWrapper>
      <ErrorMessage>{children}</ErrorMessage>
    </ErrorWrapper>
  );
};
