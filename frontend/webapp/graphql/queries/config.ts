import { gql } from '@apollo/client';

// Define the GraphQL query
export const GET_CONFIG = gql`
  query GetConfig {
    config {
      installation
    }
  }
`;
