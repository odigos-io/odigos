import { gql } from '@apollo/client';

export const UPDATE_DATA_STREAM = gql`
  mutation UpdateDataStream($id: ID!, $dataStream: DataStreamInput!) {
    updateDataStream(id: $id, dataStream: $dataStream) {
      name
    }
  }
`;

export const DELETE_DATA_STREAM = gql`
  mutation DeleteDataStream($id: ID!) {
    deleteDataStream(id: $id)
  }
`;
