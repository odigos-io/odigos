'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { StepListWrapper } from '../styled';
import { KeyvalCard } from '@/design.system';
import { StepsList } from '@/components/lists';
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
    <div style={{ height: '100vh' }}>
      <StepListWrapper>
        <StepsList currentStepIndex={1} />
      </StepListWrapper>
      <div style={{ height: '85%' }}>
        <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
          <SetupBackButton onBackClick={onBackClick} />
          <DestinationSection onSelectItem={onDestinationSelect} />
        </KeyvalCard>
      </div>
    </div>
  );
}
