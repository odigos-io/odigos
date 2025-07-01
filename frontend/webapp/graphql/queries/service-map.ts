import { gql } from '@apollo/client';

export const GET_SERVICE_MAP = gql`
  query GetServiceMap {
    getServiceMap {
      services {
        serviceName
        services {
          serviceName
          requests
          dateTime
        }
      }
    }
  }
`;
