import { gql } from '@apollo/client';

export const GET_GO_OFFSETS = gql`
  query GetGoOffsets {
    goOffsets {
      installed
      timestamp
      mods {
        module
        minVersion
        maxVersion
        minorVersions {
          minorVersion
          versions
        }
      }
    }
  }
`;
