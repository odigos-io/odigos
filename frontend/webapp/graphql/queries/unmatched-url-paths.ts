import { gql } from '@apollo/client';

export const GET_UNMATCHED_URL_PATHS = gql`
  query GetWorkloadUrlTemplatization($filter: WorkloadFilter) {
    workloads(filter: $filter) {
      urlTemplatization {
        existingConfigs {
          containerName
          templates {
            template
            examples
            notes
            actionId
            actionName
            managedBy
          }
          default {
            disabled
            skipPolicy {
              skipForNonSuccessCodes
              skipHttpStatusCodes
            }
          }
        }
        liveTrafficLearning {
          server {
            path
            count
            containerName
          }
          client {
            path
            count
            containerName
          }
          serverRecommendedRules {
            template
            reason
            segments {
              templateName
              static
              certainty
              examples
            }
          }
          clientRecommendedRules {
            template
            reason
            segments {
              templateName
              static
              certainty
              examples
            }
          }
        }
      }
    }
  }
`;
