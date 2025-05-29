'use client';

import React, { useRef } from 'react';
import { useSetupHelpers } from '@/hooks';
import { SetupHeader } from '@/components';
import { DataStreamSelectionForm, type DataStreamSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const { onClickSummary } = useSetupHelpers();
  const formRef = useRef<DataStreamSelectionFormRef>(null);

  return (
    <>
      <SetupHeader step={2} streamFormRef={formRef} />
      <DataStreamSelectionForm ref={formRef} isModal={false} onClickSummary={onClickSummary} />
    </>
  );
}
