import { gql } from '@apollo/client';

export const GET_INSTRUMENTATION_AGENTS = gql`
  query GetInstrumentationAgents {
    instrumentationAgents {
      language
      distroName
      distroDisplayName
      description
      runtimeEnvironment
      supportedRuntimeVersions
      sources
      isDefault
    }
  }
`;
