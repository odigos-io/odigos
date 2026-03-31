'use client';

import React, { useRef } from 'react';
import { useSetupHelpers } from '@/hooks';
import { OnboardingContentWrapper, SetupHeader } from '@/components';
import { DataStreamSelectionForm, type DataStreamSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const { onClickSummary } = useSetupHelpers();
  const formRef = useRef<DataStreamSelectionFormRef>(null);

  return (
    <>
      <SetupHeader step={2} streamFormRef={formRef} />
      <OnboardingContentWrapper>
        <DataStreamSelectionForm ref={formRef} isModal={false} onClickSummary={onClickSummary} />
      </OnboardingContentWrapper>
    </>
  );
}
