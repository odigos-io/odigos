import { gql } from '@apollo/client';

export const CREATE_ACTION = gql`
  mutation CreateAction($action: ActionInput!) {
    createAction(action: $action) {
      id
    }
  }
`;

export const UPDATE_ACTION = gql`
  mutation UpdateAction($id: ID!, $action: ActionInput!) {
    updateAction(id: $id, action: $action) {
      id
    }
  }
`;
