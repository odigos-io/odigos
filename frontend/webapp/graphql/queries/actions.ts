import { gql } from '@apollo/client';
export const GET_ACTIONS = gql`
  query GetActions {
    actions {
      id
      type
      spec
    }
  }
`;
