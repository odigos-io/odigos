'use client';

import React, { useRef, useState } from 'react';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-containers';

export default function Page() {
  const formRef = useRef<SourceSelectionFormRef>(null);

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));

  const { namespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  return (
    <>
      <SetupHeader entityType={ENTITY_TYPES.SOURCE} formRef={formRef} />
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
