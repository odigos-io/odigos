import { gql } from '@apollo/client';

export const GET_TOKENS = gql`
  query GetTokens {
    computePlatform {
      apiTokens {
        token
        name
        issuedAt
        expiresAt
      }
    }
  }
`;
