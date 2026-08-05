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
      title
      summary
      description
      docsUrl
      pros
      cons
      actions { type description }
    }
  }
`;
