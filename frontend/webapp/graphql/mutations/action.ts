import { gql } from '@apollo/client';

export const CREATE_ACTION = gql`
  mutation CreateAction($action: ActionInput!) {
    createAction(action: $action) {
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
`;

export const UPDATE_ACTION = gql`
  mutation UpdateAction($id: ID!, $action: ActionInput!) {
    updateAction(id: $id, action: $action) {
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
`;

export const DELETE_ACTION = gql`
  mutation DeleteAction($id: ID!, $actionType: String!) {
    deleteAction(id: $id, actionType: $actionType)
  }
`;
