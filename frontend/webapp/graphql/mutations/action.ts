import { gql } from '@apollo/client';

export const CREATE_ACTION = gql`
  mutation CreateAction($action: ActionInput!) {
    createAction(action: $action) {
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
`;

export const UPDATE_ACTION = gql`
  mutation UpdateAction($id: ID!, $action: ActionInput!) {
    updateAction(id: $id, action: $action) {
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
`;

export const DELETE_ACTION = gql`
  mutation DeleteAction($id: ID!, $actionType: String!) {
    deleteAction(id: $id, actionType: $actionType)
  }
`;
