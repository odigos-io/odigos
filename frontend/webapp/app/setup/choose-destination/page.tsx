'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { SideMenuWrapper } from '../styled';
import { ChooseDestinationContainer } from '@/containers/main';

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
