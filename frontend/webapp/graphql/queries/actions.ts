import { gql } from '@apollo/client';

export const GET_ACTIONS = gql`
  query GetActions {
    computePlatform {
      actions {
        id
        type
        name
        notes
        disabled
        signals
        fields {
          collectContainerAttributes
          collectReplicaSetAttributes
          collectWorkloadId
          collectClusterId
          labelsAttributes {
            labelKey
            attributeKey
            from
            fromSources
          }
          annotationsAttributes {
            annotationKey
            attributeKey
            from
            fromSources
          }
          clusterAttributes {
            attributeName
            attributeStringValue
          }
          overwriteExistingValues
          attributeNamesToDelete
          renames
          piiCategories

          samplingPercentage
          fallbackSamplingRatio
          endpointsFilters {
            httpRoute
            serviceName
            minimumLatencyThreshold
            fallbackSamplingRatio
          }
          servicesNameFilters {
            serviceName
            samplingRatio
            fallbackSamplingRatio
          }
          attributeFilters {
            serviceName
            attributeKey
            fallbackSamplingRatio
            condition {
              stringCondition {
                operation
                expectedValue
              }
            }
          }
        }
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
      }
    }
  }
`;
