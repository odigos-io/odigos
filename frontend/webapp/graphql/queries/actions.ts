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
        }
        conditions {
          status
          type
          reason
          message
        }
      }
    }
  }
`;
