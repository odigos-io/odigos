import { gql } from '@apollo/client';

export const GET_SERVICE_MAP = gql`
  query GetServiceMap {
    getServiceMap {
      services {
        nodeId
        serviceName
        services {
          nodeId
          isVirtual
          serviceName
          requests
          dateTime
        }
      }
    }
  }
`;
