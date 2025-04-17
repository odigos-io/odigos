'use client';

import React, { useRef, useState } from 'react';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

export default function Page() {
  const formRef = useRef<SourceSelectionFormRef>(null);

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));

  const { namespace } = useNamespace(selectedNamespace);

  return (
    <>
      <SetupHeader entityType={EntityTypes.Source} formRef={formRef} />
      <SourceSelectionForm ref={formRef} isModal={false} namespace={namespace} selectedNamespace={selectedNamespace} onSelectNamespace={onSelectNamespace} />
    </>
  );
}
