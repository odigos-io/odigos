import { gql } from '@apollo/client';

export const GET_DATA_STREAMS = gql`
  query GetDataStreams {
    dataStreams {
      name
    }
  }
`;
