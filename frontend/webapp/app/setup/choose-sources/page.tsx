'use client';
import React from 'react';

import { useSuspenseQuery, gql } from '@apollo/client';

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

  return <></>;
}
