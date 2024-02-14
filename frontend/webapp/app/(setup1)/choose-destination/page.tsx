'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { StepListWrapper } from '../styled';
import { KeyvalCard } from '@/design.system';
import { StepsList } from '@/components/lists';
import { ChooseDestinationHeader } from '@/components/setup/headers';
import { DestinationSection } from '@/containers/setup/destination/destination.section';

export default function ChooseDestinationPage() {
  const router = useRouter();

  function onDestinationSelect(type: string) {
    router.push(`/connect-destination?type=${type}`);
  }
  const cardHeaderBody = () => <ChooseDestinationHeader />;

  return (
    <>
      <StepListWrapper>
        <StepsList currentStepIndex={0} />
      </StepListWrapper>
      <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
        <div style={{ padding: '12px 40px' }}>
          <DestinationSection onSelectItem={onDestinationSelect} />
        </div>
      </KeyvalCard>
    </>
  );
}
