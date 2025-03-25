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

  const { namespaces, namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  return (
    <>
      <SetupHeader entityType={EntityTypes.Source} formRef={formRef} />
      <SourceSelectionForm
        ref={formRef}
        componentType='FAST'
        isModal={false}
        namespaces={namespaces}
        namespace={namespace}
        namespacesLoading={nsLoad}
        selectedNamespace={selectedNamespace}
        onSelectNamespace={onSelectNamespace}
      />
    </>
  );
}
