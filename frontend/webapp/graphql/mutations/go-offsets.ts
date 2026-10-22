import { gql } from '@apollo/client';

export const UPDATE_GO_OFFSETS = gql`
  mutation UpdateGoOffsets($content: String!) {
    updateGoOffsets(content: $content)
  }
`;

export const CHECK_GO_OFFSETS_UPDATES = gql`
  mutation CheckGoOffsetsUpdates($content: String!) {
    checkGoOffsetsUpdates(content: $content) {
      hasUpdates
      currentTimestamp
      proposedTimestamp
      mods {
        module
        isNew
        isRemoved
        minVersion
        maxVersion
        minorVersions {
          minorVersion
          isNew
          isRemoved
          versions {
            version
            isNew
            isRemoved
          }
        }
      }
    }
  }
`;
