'use client';
import React from 'react';
import styled from 'styled-components';
import { FlexColumn } from '@odigos/ui-components';

const LayoutContainer = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

export default function MainLayout({ children }: { children: React.ReactNode }) {
  return <LayoutContainer>{children}</LayoutContainer>;
}
