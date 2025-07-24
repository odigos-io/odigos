import { gql } from '@apollo/client';

export const UPDATE_ODIGOS_CONFIG = gql`
  mutation UpdateOdigosConfig($odigosConfig: OdigosConfigurationInput!) {
    updateOdigosConfig(odigosConfig: $odigosConfig)
  }
`;
