import { gql } from '@apollo/client';

export const GET_ODIGLET_PODS_WITH_METRICS = gql`
  query GetOdigletPodsWithMetrics {
    odigletPods {
      name
      namespace
      ready
      status
      restartsCount
      nodeName
      creationTimestamp
      image
      collectorMetrics {
        metricsAcceptedRps
        metricsDroppedRps
        exporterSuccessRps
        exporterFailedRps
        window
        lastScrape
      }
    }
  }
`;


