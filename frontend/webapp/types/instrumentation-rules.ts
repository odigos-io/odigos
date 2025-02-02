import { INSTRUMENTATION_RULE_TYPE, type WorkloadId } from '@odigos/ui-components';

export enum PayloadCollectionType {
  HTTP_REQUEST = 'httpRequest',
  HTTP_RESPONSE = 'httpResponse',
  DB_QUERY = 'dbQuery',
  MESSAGING = 'messaging',
}
export enum CodeAttributesType {
  COLUMN = 'column',
  FILE_PATH = 'filePath',
  FUNCTION = 'function',
  LINE_NUMBER = 'lineNumber',
  NAMESPACE = 'namespace',
  STACKTRACE = 'stacktrace',
}

export interface InstrumentationRuleInput {
  ruleName: string;
  notes: string;
  disabled: boolean;
  workloads: WorkloadId[] | null;
  instrumentationLibraries: InstrumentationLibraryInput[] | null;
  payloadCollection?: PayloadCollectionInput;
  codeAttributes?: CodeAttributesInput;
}

export interface InstrumentationRuleSpec {
  ruleId: string;
  ruleName: string;
  type?: INSTRUMENTATION_RULE_TYPE; // does not come from backend, it's derived during GET
  notes: string;
  disabled: boolean;
  mutable: boolean;
  profileName: string;
  workloads?: WorkloadId[];
  instrumentationLibraries?: InstrumentationLibraryGlobalId[];
  payloadCollection?: PayloadCollection;
  codeAttributes?: CodeAttributes;
}

export interface InstrumentationRuleSpecMapped extends InstrumentationRuleSpec {
  type: INSTRUMENTATION_RULE_TYPE; // does not come from backend, it's derived during GET
}

// Common types for Instrumentation Rules
enum SpanKind {
  Internal = 'Internal',
  Server = 'Server',
  Client = 'Client',
  Producer = 'Producer',
  Consumer = 'Consumer',
}
enum ProgrammingLanguage {
  Unspecified = 'Unspecified',
  Java = 'Java',
  Go = 'Go',
  JavaScript = 'JavaScript',
  Python = 'Python',
  DotNet = 'DotNet',
}
interface InstrumentationLibraryInput {
  name: string;
  spanKind?: SpanKind;
  language?: ProgrammingLanguage;
}
interface InstrumentationLibraryGlobalId {
  language: string;
  library: string;
}

// Payload Collection for Instrumentation Rules
interface PayloadCollectionInput {
  [PayloadCollectionType.HTTP_REQUEST]: {} | null;
  [PayloadCollectionType.HTTP_RESPONSE]: {} | null;
  [PayloadCollectionType.DB_QUERY]: {} | null;
  [PayloadCollectionType.MESSAGING]: {} | null;
}
export interface PayloadCollection {
  [PayloadCollectionType.HTTP_REQUEST]?: HttpPayloadCollection;
  [PayloadCollectionType.HTTP_RESPONSE]?: HttpPayloadCollection;
  [PayloadCollectionType.DB_QUERY]?: DbQueryPayloadCollection;
  [PayloadCollectionType.MESSAGING]?: MessagingPayloadCollection;
}
interface HttpPayloadCollection {
  mimeTypes?: string[];
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}
interface DbQueryPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}
interface MessagingPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

// Code Attributes for Instrumentation Rules
interface CodeAttributesInput {
  [CodeAttributesType.COLUMN]: boolean | null;
  [CodeAttributesType.FILE_PATH]: boolean | null;
  [CodeAttributesType.FUNCTION]: boolean | null;
  [CodeAttributesType.LINE_NUMBER]: boolean | null;
  [CodeAttributesType.NAMESPACE]: boolean | null;
  [CodeAttributesType.STACKTRACE]: boolean | null;
}
interface CodeAttributes {
  [CodeAttributesType.COLUMN]?: boolean;
  [CodeAttributesType.FILE_PATH]?: boolean;
  [CodeAttributesType.FUNCTION]?: boolean;
  [CodeAttributesType.LINE_NUMBER]?: boolean;
  [CodeAttributesType.NAMESPACE]?: boolean;
  [CodeAttributesType.STACKTRACE]?: boolean;
}
