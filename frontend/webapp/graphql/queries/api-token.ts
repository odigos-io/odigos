import { gql } from '@apollo/client';

export const GET_API_TOKENS = gql`
  query GetApiTokens {
    getApiTokens {
      token
      aud
      iat
      exp
    }
  }
`;
