'use client';

import React, { useRef } from 'react';
import { useNamespace, useSetupHelpers } from '@/hooks';
import { OnboardingContentWrapper, SetupHeader } from '@/components';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { onClickSummary } = useSetupHelpers();
  const formRef = useRef<SourceSelectionFormRef>(null);

  return (
    <>
      <SetupHeader step={3} sourceFormRef={formRef} />
      <OnboardingContentWrapper>
        <SourceSelectionForm ref={formRef} isModal={false} fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads} onClickSummary={onClickSummary} />
      </OnboardingContentWrapper>
    </>
  );
}
