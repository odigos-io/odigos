import { gql } from '@apollo/client';

export const GET_ACTIONS = gql`
  query GetActions {
    computePlatform {
      actions {
        id
        type
        spec
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
      }
    }
  }
`;
