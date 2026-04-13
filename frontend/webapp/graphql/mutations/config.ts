import { gql } from '@apollo/client';

export const UPDATE_LOCAL_UI_CONFIG = gql`
  mutation UpdateLocalUiConfig($config: LocalUiConfigInput!) {
    updateLocalUiConfig(config: $config)
  }
`;
