import { gql } from '@apollo/client';

export const UPDATE_DATA_STREAM = gql`
  mutation UpdateDataStream($dataStreamName: String!, $dataStream: DataStreamInput!) {
    updateDataStream(dataStreamName: $dataStreamName, dataStream: $dataStream) {
      name
    }
  }
`;

export const DELETE_DATA_STREAM = gql`
  mutation DeleteDataStream($dataStreamName: String!) {
    deleteDataStream(dataStreamName: $dataStreamName)
  }
`;
