'use client';
import React from 'react';
import styled from 'styled-components';
import { SideMenu } from '@/components';
import { ChooseSourcesContainer } from '@/containers/main';

const SideMenuWrapper = styled.div`
  position: absolute;
  left: 24px;
  top: 144px;
`;

export default function ChooseSourcesPage() {
  return (
    <>
      <SideMenuWrapper>
        <SideMenu currentStep={2} />
      </SideMenuWrapper>
      <ChooseSourcesContainer />
    </>
  );
}
