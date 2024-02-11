'use client';
import { DestinationSection } from '@/containers/setup/destination/destination.section';
import { KeyvalCard } from '@/design.system';
import React from 'react';
import { StepListWrapper } from '../styled';
import { StepsList } from '@/components/lists';
import { useNotification } from '@/hooks';

export default function ChooseDestinationPage() {
  const { show, Notification } = useNotification();

  return (
    <>
      <StepListWrapper>
        <StepsList currentStepIndex={0} />
      </StepListWrapper>
      <KeyvalCard type={'secondary'} header={{ body: () => <div></div> }}>
        <div style={{ padding: '0 40px' }}>
          <DestinationSection />
        </div>
      </KeyvalCard>
      <Notification />
    </>
  );
}
