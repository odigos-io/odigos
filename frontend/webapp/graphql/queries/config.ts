import { gql } from '@apollo/client';

export const GET_CONFIG = gql`
  query GetConfig {
    config {
      readonly
      platformType
      tier
      odigosVersion
      installationMethod
      installationStatus
      clusterName
      isCentralProxyRunning
    }
  }
`;

export const GET_CONFIG_YAMLS = gql`
  query GetConfigYamls {
    configYamls {
      configs {
        name
        displayName
        fields {
          displayName
          componentType
          isHelmOnly
          description
          helmValuePath
          docsLink
          componentProps
        }
      }
    }
  }
`;
