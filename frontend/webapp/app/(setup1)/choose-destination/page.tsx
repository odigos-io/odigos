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
    <>
      <StepListWrapper>
        <StepsList currentStepIndex={1} />
      </StepListWrapper>
      <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
        <SetupBackButton onBackClick={onBackClick} />
        <div style={{ padding: '12px 40px' }}>
          <DestinationSection onSelectItem={onDestinationSelect} />
        </div>
      </KeyvalCard>
    </>
  );
}
