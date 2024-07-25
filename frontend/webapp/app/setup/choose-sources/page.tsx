'use client';
import React from 'react';

import { useSuspenseQuery, gql } from '@apollo/client';
import { Input, SectionTitle } from '@/reuseable-components';

const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform($cpId: ID!) {
    computePlatform(cpId: $cpId) {
      id
      name
      computePlatformType
      k8sActualSources {
        namespace
        kind
        name
        serviceName
        autoInstrumented
        creationTimestamp
        numberOfInstances
        hasInstrumentedApplication
        instrumentedApplicationDetails {
          languages {
            containerName
            language
          }
          conditions {
            type
            status
            lastTransitionTime
            reason
            message
          }
        }
      }
    }
  }
`;

export default function ChooseSourcesPage() {
  const { error, data } = useSuspenseQuery(GET_COMPUTE_PLATFORM, {
    variables: { cpId: '1' },
  });

  return (
    <div style={{ width: 800 }}>
      <SectionTitle
        title="Choose sources"
        description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations."
      />
      <Input
        placeholder="Search for sources"
        icon={'/icons/common/search.svg'}
      />
    </div>
  );
}
