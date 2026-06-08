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
        urlTemplatizationRulesGroups {
          filterK8sNamespace
          filterK8sWorkloadKind
          filterK8sWorkloadName
          filterProgrammingLanguage
          notes
          workloadFilters {
            kind
            name
          }
          templatizationRules {
            template
            notes
            examples
          }
        }
        extractAttribute {
          extractions {
            targetAttributeName
            lookupKey
            dataFormat
            regex
          }
        }
      }
      conditions {
        status
        type
        reason
        message
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
        urlTemplatizationRulesGroups {
          filterK8sNamespace
          filterK8sWorkloadKind
          filterK8sWorkloadName
          filterProgrammingLanguage
          notes
          workloadFilters {
            kind
            name
          }
          templatizationRules {
            template
            notes
            examples
          }
        }
        extractAttribute {
          extractions {
            targetAttributeName
            lookupKey
            dataFormat
            regex
          }
        }
      }
      conditions {
        status
        type
        reason
        message
      }
    }
  }
`;

export const DELETE_ACTION = gql`
  mutation DeleteAction($id: ID!, $actionType: String!) {
    deleteAction(id: $id, actionType: $actionType)
  }
`;
