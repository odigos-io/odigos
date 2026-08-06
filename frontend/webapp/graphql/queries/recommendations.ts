import { gql } from '@apollo/client';

const RECOMMENDATION_FIELDS = `
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
    applyExamples { type content }
  }
`;

export const GET_RECOMMENDATIONS = gql`
  query GetRecommendations {
    computePlatform {
      recommendations { ${RECOMMENDATION_FIELDS} }
    }
  }
`;
