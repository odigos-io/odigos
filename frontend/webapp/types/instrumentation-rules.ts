export enum InstrumentationRuleType {
  PAYLOAD_COLLECTION = 'payload-collection',
}

// Define the types for the Instrumentation Rule Spec
export interface InstrumentationRuleSpec {
  ruleName: string;
  notes?: string;
  disabled?: boolean;
  workloads?: PodWorkload[];
  instrumentationLibraries?: InstrumentationLibraryGlobalId[];
  payloadCollection?: PayloadCollection;
}

export interface PodWorkload {
  name: string;
  namespace: string;
  kind: string;
}

export interface InstrumentationLibraryGlobalId {
  language: string;
  library: string;
}

export interface PayloadCollection {
  httpRequest?: HttpPayloadCollection;
  httpResponse?: HttpPayloadCollection;
  dbQuery?: DbQueryPayloadCollection;
  messaging?: MessagingPayloadCollection;
}

export interface MessagingPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

export interface HttpPayloadCollection {
  mimeTypes?: string[];
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}

export interface DbQueryPayloadCollection {
  maxPayloadLength?: number;
  dropPartialPayloads?: boolean;
}
