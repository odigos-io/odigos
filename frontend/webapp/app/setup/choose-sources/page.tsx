'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { SideMenuWrapper } from '../styled';
import { ChooseSourcesContainer } from '@/containers/main';

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
