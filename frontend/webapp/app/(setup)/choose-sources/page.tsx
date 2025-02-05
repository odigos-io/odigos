'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { OnboardingSideMenuWrapper } from '@/styles';
import { ChooseSourcesContainer } from '@/containers/main';

export default function ChooseSourcesPage() {
  return (
    <>
      <OnboardingSideMenuWrapper>
        <SideMenu currentStep={2} />
      </OnboardingSideMenuWrapper>
      <ChooseSourcesContainer />
    </>
  );
}
