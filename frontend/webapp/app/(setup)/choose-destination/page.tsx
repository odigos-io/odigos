'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { OnboardingSideMenuWrapper } from '@/styles';
import { AddDestinationContainer } from '@/containers/main';

export default function ChooseDestinationPage() {
  return (
    <>
      <OnboardingSideMenuWrapper>
        <SideMenu currentStep={3} />
      </OnboardingSideMenuWrapper>
      <AddDestinationContainer />
    </>
  );
}
