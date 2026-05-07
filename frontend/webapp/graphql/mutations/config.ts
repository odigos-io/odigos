import { gql } from '@apollo/client';

export const UPDATE_LOCAL_UI_CONFIG = gql`
  mutation UpdateLocalUiConfig($config: LocalUiConfigInput!) {
    updateLocalUiConfig(config: $config)
  }
`;

export const RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS = gql`
  mutation ResetLocalUiConfigToFactoryDefaults {
    resetLocalUiConfigToFactoryDefaults
  }
`;
