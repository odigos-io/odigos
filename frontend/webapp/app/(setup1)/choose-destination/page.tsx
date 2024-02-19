'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { StepsList } from '@/components/lists';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';
import { DestinationSection } from '@/containers/setup/destination/destination.section';
import {
  ChooseDestinationHeader,
  SetupBackButton,
} from '@/components/setup/headers';

export default function ChooseDestinationPage() {
  const router = useRouter();

  function onDestinationSelect(type: string) {
    router.push(`/connect-destination?type=${type}`);
  }

  function onBackClick() {
    router.back();
  }

  const cardHeaderBody = () => <ChooseDestinationHeader />;

  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={1} />
      </StepListWrapper>
      <CardWrapper>
        <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
          <SetupBackButton onBackClick={onBackClick} />
          <DestinationSection onSelectItem={onDestinationSelect} />
        </KeyvalCard>
      </CardWrapper>
    </PageContainer>
  );
}
