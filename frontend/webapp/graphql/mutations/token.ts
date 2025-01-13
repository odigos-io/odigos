import { gql } from '@apollo/client';

export const UPDATE_API_TOKEN = gql`
  mutation UpdateApiToken($token: String!) {
    updateApiToken(token: $token)
  }
`;
