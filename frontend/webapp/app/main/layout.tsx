'use client';
import React, { useEffect } from 'react';
import styled from 'styled-components';
import { MainHeader } from '@/components';
import { useActualSources, useComputePlatform, useGetActions } from '@/hooks';
import { useActualDestination } from '@/hooks/destinations/useActualDestinations';

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
  const { data } = useComputePlatform();
  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();

  useEffect(() => {
    if (data || destinations) {
      console.log('data', actions, sources, destinations, data);
    }
  }, [destinations, data, actions, sources]);
  return (
    <LayoutContainer>
      <MainContent>
        <MainHeader />
        {children}
      </MainContent>
    </LayoutContainer>
  );
}
