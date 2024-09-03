'use client';
import { MainHeader } from '@/components';
import React from 'react';
import styled from 'styled-components';

const LayoutContainer = styled.div`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  display: flex;
  align-items: center;
  flex-direction: column;
`;

const MainContent = styled.div`
  display: flex;
  width: 100vw;
  height: 76px;
  flex-direction: column;
  align-items: center;
`;

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <LayoutContainer>
      <MainContent>
        <MainHeader />
        {children}
      </MainContent>
    </LayoutContainer>
  );
}
