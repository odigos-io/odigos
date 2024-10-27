import { gql } from '@apollo/client';

export const CREATE_ACTION = gql`
  mutation CreateAction($action: ActionInput!) {
    createAction(action: $action) {
      id
    }
  }
`;
