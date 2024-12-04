'use client';
import React from 'react';
import { SideMenu } from '@/components';
import { SideMenuWrapper } from '../styled';
import { AddDestinationContainer } from '@/containers/main';

export default function ChooseDestinationPage() {
  return (
    <>
      <SideMenuWrapper>
        <SideMenu currentStep={3} />
      </SideMenuWrapper>
      <AddDestinationContainer />
    </>
  );
}
