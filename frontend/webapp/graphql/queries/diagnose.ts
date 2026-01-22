import { gql } from "@apollo/client";

export const DOWNLOAD_DIAGNOSE = gql`
  query DownloadDiagnose($input: DiagnoseInput!, $dryRun: Boolean) {
    diagnose(input: $input, dryRun: $dryRun) {
      stats {
        fileCount
        totalSizeBytes
        totalSizeHuman
      }
      includeProfiles
      includeMetrics
      includeSourceWorkloads
      sourceWorkloadNamespaces
    }
  }
`;