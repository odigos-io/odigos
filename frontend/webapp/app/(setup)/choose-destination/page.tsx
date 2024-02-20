'use client';
import React from 'react';
import { StepsList } from '@/components';
import { ChooseDestinationContainer } from '@/containers';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';

export default function ChooseDestinationPage() {
  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={1} />
      </StepListWrapper>
      <CardWrapper>
        <ChooseDestinationContainer />
      </CardWrapper>
    </PageContainer>
  );
}
