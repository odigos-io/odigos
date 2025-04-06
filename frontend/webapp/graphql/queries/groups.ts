import { gql } from '@apollo/client';

export const GET_GROUP_NAMES = gql`
  query GetGroupNames {
    groupNames
  }
`;
