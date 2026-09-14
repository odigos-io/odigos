import { gql } from '@apollo/client';

export const GET_INSTRUMENTATION_AGENTS = gql`
  query GetInstrumentationAgents {
    instrumentationAgents {
      language
      distroName
      distroDisplayName
      description
      tier
      kind
      runtimeEnvironment
      supportedRuntimeVersions
      fallbackDistroNames
      instrumentedContainers
      uninstrumentedContainers
      sources
    }
  }
`;
