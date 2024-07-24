'use client';
import React from 'react';
import styled from 'styled-components';
import { SetupHeader, SideMenu } from '@/components';

const LayoutContainer = styled.div`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  display: flex;
  align-items: center;
  flex-direction: column;
`;

const SideMenuWrapper = styled.div`
  position: absolute;
  left: 24px;
  top: 144px;
`;

const MainContent = styled.div`
  display: flex;
  max-width: 1440px;
  width: 100%;
  background-color: ${({ theme }) => theme.colors.secondary};
  flex-direction: column;
  align-items: center;
`;

export default function SetupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <LayoutContainer>
      <div style={{ width: '100vw' }}>
        <SetupHeader />
      </div>
      <SideMenuWrapper>
        <SideMenu />
      </SideMenuWrapper>
      <MainContent>
        <SideMenu />

        {children}
      </MainContent>
    </LayoutContainer>
  );
}
