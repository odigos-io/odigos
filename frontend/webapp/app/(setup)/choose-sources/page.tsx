'use client';

import React, { useRef } from 'react';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const formRef = useRef<SourceSelectionFormRef>(null);
  const { fetchNamespace } = useNamespace();

  return (
    <>
      <SetupHeader step={3} sourceFormRef={formRef} />
      <SourceSelectionForm ref={formRef} isModal={false} fetchSingleNamespace={fetchNamespace} />
    </>
  );
}
