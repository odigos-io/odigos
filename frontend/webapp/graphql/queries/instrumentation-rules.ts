import { gql } from '@apollo/client';

export const GET_INSTRUMENTATION_RULE_TYPES = gql`
  query GetInstrumentationRuleTypes {
    ruleTypes {
      type
      displayName
      description
      supportedLanguages
      docsUrl
      fields {
        name
        displayName
        componentType
        componentProperties
        initialValue
        renderCondition
      }
    }
  }
`;

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
        }
        sourcesScopes {
          workloadName
          workloadKind
          workloadNamespace
          workloadLanguage
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
          golang {
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
