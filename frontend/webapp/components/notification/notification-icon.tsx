import React from 'react';
import styled from 'styled-components';
import { BlueInfo, GreenCheck, RedError } from '@/assets/icons/app';

interface IconWrapperProps {
  bgColor: string;
  borderColor: string;
  children: React.ReactNode;
}

const IconWrapper = styled.div<{ bgColor: string }>`
  width: 32px;
  height: 32px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: ${({ bgColor }) => bgColor};
`;

const InnerIconWrapper = styled.div<{ borderColor: string }>`
  width: 16px;
  height: 16px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid ${({ borderColor }) => borderColor};
`;

export const SuccessIcon = () => (
  <IconWrapper bgColor="#3fb94f40">
    <InnerIconWrapper borderColor="#3fb950">
      <GreenCheck style={{ width: 10, height: 10 }} />
    </InnerIconWrapper>
  </IconWrapper>
);

export const ErrorIcon = () => (
  <IconWrapper bgColor="#f8524952">
    <InnerIconWrapper borderColor="#f85249">
      <RedError
        style={{ width: 10, height: 10, marginLeft: 2, marginBottom: 2 }}
      />
    </InnerIconWrapper>
  </IconWrapper>
);

export const InfoIcon = () => (
  <IconWrapper bgColor="#2196F340">
    <InnerIconWrapper borderColor="#2196F3">
      <BlueInfo />
    </InnerIconWrapper>
  </IconWrapper>
);

export const getIcon = (type) => {
  switch (type) {
    case 'success':
      return <SuccessIcon />;
    case 'error':
      return <ErrorIcon />;
    case 'info':
      return <InfoIcon />;
    default:
      return null;
  }
};
