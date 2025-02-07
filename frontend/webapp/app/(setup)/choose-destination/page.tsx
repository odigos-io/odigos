'use client';
import React from 'react';
import { useSSE } from '@/hooks';
import { SideMenu } from '@/components';
import { OnboardingSideMenuWrapper } from '@/styles';
import { AddDestinationContainer } from '@/containers/main';

export default function ChooseDestinationPage() {
  // call important hooks that should run on page-mount
  useSSE();

  return (
    <>
      <OnboardingSideMenuWrapper>
        <SideMenu currentStep={3} />
      </OnboardingSideMenuWrapper>
      <AddDestinationContainer />
    </>
  );
}
