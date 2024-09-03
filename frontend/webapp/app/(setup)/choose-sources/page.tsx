'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { ChooseSourcesContainer } from '@/containers/main';
import { SideMenuWrapper } from '../styled';

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
