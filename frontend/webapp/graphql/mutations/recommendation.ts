import { gql } from '@apollo/client';

export const SET_RECOMMENDATION_DISMISSED = gql`
  mutation SetRecommendationDismissed($name: ID!, $dismissed: Boolean!) {
    setRecommendationDismissed(name: $name, dismissed: $dismissed) {
      name
      type
      applied
      conditionsMet
      dismissed
      oss
      requireOdigosDeployment
      catalogConditions { type }
      appliedWhen { type expression actionType }
      categories
      title
      summary
      description
      docsUrl
      pros
      cons
      remediations {
        type
        buttonText
        tooltip
        canApplyViaUi
        applyExamples { type content }
      }
    }
  }
`;

export const APPLY_RECOMMENDATION_REMEDIATION = gql`
  mutation ApplyRecommendationRemediation($recommendationType: String!, $remediationType: String!) {
    applyRecommendationRemediation(recommendationType: $recommendationType, remediationType: $remediationType)
  }
`;
