'use client';
import React from 'react';
import styled from 'styled-components';
import { SideMenu } from '@/components';
import { ChooseDestinationContainer } from '@/containers/main';

const SideMenuWrapper = styled.div`
  position: absolute;
  left: 24px;
  top: 144px;
`;

export default function ChooseDestinationPage() {
  return (
    <>
      <SideMenuWrapper>
        <SideMenu currentStep={3} />
      </SideMenuWrapper>
      <ChooseDestinationContainer />
    </>
  );
}
