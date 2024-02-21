'use client';
import { StepsList } from '@/components';
import { ConnectDestinationContainer } from '@/containers';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';

export default function ConnectDestinationPage() {
  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={2} />
      </StepListWrapper>
      <CardWrapper>
        <ConnectDestinationContainer />
      </CardWrapper>
    </PageContainer>
  );
}
