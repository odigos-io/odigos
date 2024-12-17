// Enumeration of possible Instrumentation Rule Types
export enum InstrumentationRuleType {
  UNKNOWN_TYPE = 'UnknownType',
  PAYLOAD_COLLECTION = 'PayloadCollection',
}

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

enum K8sResourceKind {
  Deployment = 'Deployment',
  DaemonSet = 'DaemonSet',
  StatefulSet = 'StatefulSet',
}

interface PayloadCollectionInput {
  httpRequest: {
    // mimeTypes: string[];
    // maxPayloadLength: number;
    // dropPartialPayloads: boolean;
  } | null;
  httpResponse: {
    // mimeTypes: string[];
    // maxPayloadLength: number;
    // dropPartialPayloads: boolean;
  } | null;
  dbQuery: {
    // maxPayloadLength: number;
    // dropPartialPayloads: boolean;
  } | null;
  messaging: {
    // maxPayloadLength: number;
    // dropPartialPayloads: boolean;
  } | null;
}

export interface InstrumentationRuleInput {
  ruleName: string;
  notes: string;
  disabled: boolean;
  workloads: PodWorkload[] | null;
  instrumentationLibraries: InstrumentationLibraryInput[] | null;
  payloadCollection: PayloadCollectionInput;
}

// delete this ? (used by old-UI)
export enum RulesType {
  ADD_METADATA = 'add-metadata',
  ERROR_SAMPLING = 'error-sampling',
  PII_MASKING = 'pii-masking',
  PAYLOAD_COLLECTION = 'payload-collection',
}

export enum RulesSortType {
  TYPE = 'type',
  RULE_NAME = 'ruleName',
  STATUS = 'status',
}

export interface InstrumentationRuleSpec {
  ruleId: string;
  ruleName: string;
  type?: InstrumentationRuleType; // does not come from backend, it's derived during GET
  notes: string;
  disabled: boolean;
  workloads?: PodWorkload[];
  instrumentationLibraries?: InstrumentationLibraryGlobalId[];
  payloadCollection?: PayloadCollection;
}

export interface InstrumentationRuleSpecMapped extends InstrumentationRuleSpec {
  type: InstrumentationRuleType; // does not come from backend, it's derived during GET
}

// Definition of a Pod Workload type
export interface PodWorkload {
  name: string;
  namespace: string;
  kind: K8sResourceKind;
}

// Definition of Instrumentation Library Global ID
export interface InstrumentationLibraryGlobalId {
  language: string;
  library: string;
}

export interface InstrumentationLibraryInput {
  name: string;
  spanKind?: SpanKind;
  language?: ProgrammingLanguage;
}

export enum PayloadCollectionType {
  HTTP_REQUEST = 'httpRequest',
  HTTP_RESPONSE = 'httpResponse',
  DB_QUERY = 'dbQuery',
  MESSAGING = 'messaging',
}

// Payload Collection Interface for Instrumentation Rules
export interface PayloadCollection {
  [PayloadCollectionType.HTTP_REQUEST]?: HttpPayloadCollection;
  [PayloadCollectionType.HTTP_RESPONSE]?: HttpPayloadCollection;
  [PayloadCollectionType.DB_QUERY]?: DbQueryPayloadCollection;
  [PayloadCollectionType.MESSAGING]?: MessagingPayloadCollection;
}

// Messaging Payload Collection Interface
export interface MessagingPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

// HTTP Payload Collection Interface
export interface HttpPayloadCollection {
  mimeTypes?: string[];
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

// Database Query Payload Collection Interface
export interface DbQueryPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

// Interface for Metadata addition rules
export interface MetadataAddition {
  metadata: { [key: string]: string };
}

// Interface for Error Sampling rules
export interface ErrorSampling {
  samplingRatio: number;
}

// Interface for PII Masking rules
export interface PIIMasking {
  piiCategories: string[];
}
