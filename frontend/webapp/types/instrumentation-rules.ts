// Enumeration of possible Instrumentation Rule Types
export enum InstrumentationRuleType {
  PAYLOAD_COLLECTION = 'payload-collection',
}

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

// Define the types for the Instrumentation Rule Spec
export interface InstrumentationRuleSpec {
  ruleId?: string;
  ruleName: string;
  notes?: string;
  disabled?: boolean;
  workloads?: PodWorkload[];
  instrumentationLibraries?: InstrumentationLibraryGlobalId[];
  payloadCollection?: PayloadCollection;
}

// Definition of a Pod Workload type
export interface PodWorkload {
  name: string;
  namespace: string;
  kind: string;
}

// Definition of Instrumentation Library Global ID
export interface InstrumentationLibraryGlobalId {
  language: string;
  library: string;
}

// Payload Collection Interface for Instrumentation Rules
export interface PayloadCollection {
  httpRequest?: HttpPayloadCollection;
  httpResponse?: HttpPayloadCollection;
  dbQuery?: DbQueryPayloadCollection;
  messaging?: MessagingPayloadCollection;
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
