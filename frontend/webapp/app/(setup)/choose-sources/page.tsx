'use client';

import React, { useRef, useState } from 'react';
import { useNamespace, useSSE } from '@/hooks';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { Stepper } from '@odigos/ui-components';
import { OnboardingStepperWrapper } from '@/styles';
import SetupHeader from '@/components/lib-imports/setup-header';
import { FormRef, SourceSelectionForm } from '@odigos/ui-containers';

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  const formRef = useRef<FormRef>(null);

  return (
    <>
      <SetupHeader entityType={ENTITY_TYPES.SOURCE} formRef={formRef} />

      <OnboardingStepperWrapper>
        <Stepper
          currentStep={2}
          data={[
            { stepNumber: 1, title: 'INSTALLATION' },
            { stepNumber: 2, title: 'SOURCES' },
            { stepNumber: 3, title: 'DESTINATIONS' },
          ]}
        />
      </OnboardingStepperWrapper>

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
