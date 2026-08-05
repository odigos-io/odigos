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
  title
  summary
  description
  docsUrl
  pros
  cons
  actions { type description }
`;

export const GET_RECOMMENDATIONS = gql`
  query GetRecommendations {
    computePlatform {
      recommendations { ${RECOMMENDATION_FIELDS} }
    }
  }
`;
