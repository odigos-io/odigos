'use client';
import React from 'react';
import { StepsList } from '@/components/lists';
import { ChooseSourcesContainer } from '@/containers';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';

export default function ChooseSourcesPage() {
  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={0} />
      </StepListWrapper>
      <CardWrapper>
        <ChooseSourcesContainer />
      </CardWrapper>
    </PageContainer>
  );
}
