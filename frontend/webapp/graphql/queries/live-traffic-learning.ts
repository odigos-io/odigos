import { gql } from '@apollo/client';

export const GET_LIVE_TRAFFIC_LEARNING = gql`
  query GetWorkloadLiveTrafficLearning($filter: WorkloadFilter) {
    workloads(filter: $filter) {
      containers {
        containerName
        urlTemplatization {
          templatization {
            templates {
              template
              examples
              notes
              actionId
              actionName
              managedBy
            }
            patternBasedTemplatization {
              disabled
              skipPolicy {
                skipForNonSuccessCodes
                skipHttpStatusCodes
              }
            }
          }
          liveTrafficLearning {
            unmatchedServerPaths {
              path
              count
            }
            unmatchedClientPaths {
              path
              count
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
  }
`;
