'use client';
import React from 'react';
import { useSSE } from '@/hooks';
import { SideMenu } from '@/components';
import { OnboardingSideMenuWrapper } from '@/styles';
import { ChooseSourcesContainer } from '@/containers/main';

export default function ChooseSourcesPage() {
  // call important hooks that should run on page-mount
  useSSE();

  return (
    <>
      <OnboardingSideMenuWrapper>
        <SideMenu currentStep={2} />
      </OnboardingSideMenuWrapper>
      <ChooseSourcesContainer />
    </>
  );
}
