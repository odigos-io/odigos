'use client';

import React, { useRef } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { SetupHeader } from '@/components';
import { ROUTES, SKIP_TO_SUMMERY_QUERY_PARAM } from '@/utils';
import { DataStreamSelectionForm, type DataStreamSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const router = useRouter();
  const params = useSearchParams();
  const skipToSummary = !!params.get(SKIP_TO_SUMMERY_QUERY_PARAM);

  const formRef = useRef<DataStreamSelectionFormRef>(null);

  return (
    <>
      <SetupHeader step={2} streamFormRef={formRef} />
      <DataStreamSelectionForm ref={formRef} isModal={false} onClickSummary={skipToSummary ? () => router.push(ROUTES.SETUP_SUMMARY) : undefined} />
    </>
  );
}
