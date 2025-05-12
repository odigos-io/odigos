'use client';

import React, { useRef } from 'react';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const formRef = useRef<SourceSelectionFormRef>(null);
  const { fetchNamespace } = useNamespace();

  return (
    <>
      <SetupHeader entityType={EntityTypes.Source} formRef={formRef} />
      <SourceSelectionForm ref={formRef} isModal={false} fetchSingleNamespace={fetchNamespace} />
    </>
  );
}
