import { gql } from '@apollo/client';

export const GET_PEER_SOURCES = gql`
  query GetPeerSources($serviceName: String!) {
    peerSources(serviceName: $serviceName) {
      inbound {
        serviceName
        requests
        dateTime
      }
      outbound {
        serviceName
        requests
        dateTime
      }
    }
  }
`;
