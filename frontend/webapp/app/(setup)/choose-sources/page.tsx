'use client';

import React, { useRef, useState } from 'react';
import { useNamespace } from '@/hooks';
import { SetupHeader } from '@/components';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { SourceSelectionForm, type SourceSelectionFormRef } from '@odigos/ui-containers';

export default function Page() {
  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);
  const formRef = useRef<SourceSelectionFormRef>(null);

  return (
    <>
      <SetupHeader entityType={ENTITY_TYPES.SOURCE} formRef={formRef} />
      <SourceSelectionForm
        ref={formRef}
        componentType='FAST'
        isModal={false}
        namespaces={allNamespaces}
        namespace={namespace}
        namespacesLoading={nsLoad}
        selectedNamespace={selectedNamespace}
        onSelectNamespace={onSelectNamespace}
      />
    </>
  );
}
