'use client';

import React, { useRef } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ROUTES } from '@/utils';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const router = useRouter();
  const params = useSearchParams();
  const skipToSummary = !!params.get('skipToSummary');

  const { fetchNamespace } = useNamespace();
  const formRef = useRef<SourceSelectionFormRef>(null);

  return (
    <>
      <SetupHeader step={3} sourceFormRef={formRef} />
      <SourceSelectionForm ref={formRef} isModal={false} fetchSingleNamespace={fetchNamespace} onClickSummary={skipToSummary ? () => router.push(ROUTES.SETUP_SUMMARY) : undefined} />
    </>
  );
}
