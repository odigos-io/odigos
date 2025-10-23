import { gql } from '@apollo/client';

export const GET_INSTRUMENTATION_RULES = gql`
  query GetInstrumentationRules {
    computePlatform {
      instrumentationRules {
        type
        ruleId
        ruleName
        notes
        disabled
        mutable
        profileName
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
        payloadCollection {
          httpRequest {
            mimeTypes
            maxPayloadLength
            dropPartialPayloads
          }
          httpResponse {
            mimeTypes
            maxPayloadLength
            dropPartialPayloads
          }
          dbQuery {
            maxPayloadLength
            dropPartialPayloads
          }
          messaging {
            maxPayloadLength
            dropPartialPayloads
          }
        }
        codeAttributes {
          column
          filePath
          function
          lineNumber
          namespace
          stacktrace
        }
        headersCollection {
          headerKeys
        }
        customInstrumentations {
          golang{
            packageName
            functionName
            receiverName
            receiverMethodName
          }
          java {
            methodName
            className
          }
        }
      }
    }
  }
`;
