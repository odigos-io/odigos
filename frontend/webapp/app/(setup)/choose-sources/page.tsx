'use client';

import React, { useRef, useState } from 'react';
import { useNamespace } from '@/hooks';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { Stepper } from '@odigos/ui-components';
import { OnboardingStepperWrapper } from '@/components';
import SetupHeader from '@/components/lib-imports/setup-header';
import PageContainer from '@/components/providers/page-container';
import { SourceSelectionForm, ToastList, type SourceSelectionFormRef } from '@odigos/ui-containers';

export default function Page() {
  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);
  const formRef = useRef<SourceSelectionFormRef>(null);

  return (
    <PageContainer>
      <ToastList />
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
    </PageContainer>
  );
}
