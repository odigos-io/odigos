import { gql } from '@apollo/client';

export const PERSIST_SOURCES = gql`
  mutation PersistSources($sources: [PersistNamespaceSourceInput!]!) {
    persistK8sSources(sources: $sources)
  }
`;

export const UPDATE_K8S_ACTUAL_SOURCE = gql`
  mutation UpdateK8sActualSource($sourceId: K8sSourceId!, $patchSourceRequest: PatchSourceRequestInput!) {
    updateK8sActualSource(sourceId: $sourceId, patchSourceRequest: $patchSourceRequest)
  }
`;
