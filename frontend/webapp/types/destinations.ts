import { Condition } from './common';

export enum DestinationsSortType {
  NAME = 'name',
  TYPE = 'type',
}

interface ObservabilitySignalSupport {
  supported: boolean;
}

interface SupportedSignals {
  logs: ObservabilitySignalSupport;
  metrics: ObservabilitySignalSupport;
  traces: ObservabilitySignalSupport;
}

export interface DestinationTypeItem {
  type: string;
  testConnectionSupported: boolean;
  displayName: string;
  imageUrl: string;
  supportedSignals: SupportedSignals;
  fields: {
    [key: string]: string;
  };
}

export interface DestinationsCategory {
  name: string;
  items: DestinationTypeItem[];
}

export interface GetDestinationTypesResponse {
  destinationTypes: {
    categories: DestinationsCategory[];
  };
}

export interface DestinationDetailsField {
  name: string;
  displayName: string;
  componentType: string;
  componentProperties: string;
  videoUrl: string | null;
  thumbnailURL: string | null;
  initialValue: string;
  __typename: string;
}

export type DynamicField = {
  name: string;
  componentType: 'input' | 'dropdown' | 'multi_input' | 'textarea';
  title: string;
  [key: string]: any;
};

export interface DestinationDetailsResponse {
  destinationTypeDetails: {
    fields: DestinationDetailsField[];
  };
}

export interface ExportedSignals {
  logs: boolean;
  metrics: boolean;
  traces: boolean;
}

interface FieldInput {
  key: string;
  value: string;
}

export interface DestinationInput {
  name: string;
  type: string;
  exportedSignals: ExportedSignals;
  fields: FieldInput[];
}

export type DestinationTypeDetail = {
  title: string;
  value: string;
};

export type ConfiguredDestination = {
  displayName: string;
  category: string;
  type: string;
  exportedSignals: ExportedSignals;
  imageUrl: string;
  destinationTypeDetails: DestinationTypeDetail[];
};

export interface DestinationType {
  fields: any;
  display_name: string;
  image_url: string;
  id: string;
}

interface SupportedSignal {
  supported: boolean;
}

export interface SupportedDestinationSignals {
  traces: SupportedSignal;
  metrics: SupportedSignal;
  logs: SupportedSignal;
}

export interface SelectedDestination {
  type: string;
  display_name: string;
  image_url: string;
  supported_signals: SupportedDestinationSignals;
  test_connection_supported: boolean;
}

export interface Destination {
  id: string;
  name: string;
  type: string;
  signals: {
    traces: boolean;
    metrics: boolean;
    logs: boolean;
  };
  fields: Record<string, any>;
  conditions: Condition[];
  destination_type: {
    type: string;
    display_name: string;
    image_url: string;
    supported_signals: SupportedDestinationSignals;
  };
}

export interface Field {
  name: string;
  component_type: string;
  display_name: string;
  component_properties: any;
  video_url: string;
  initial_value?: string;
}

export interface DestinationConfig {
  type: string;
  name: string;
  signals: SupportedDestinationSignals;
  fields: {
    [key: string]: string;
  };
}

export interface ActualDestination {
  id: string;
  name: string;
  type: string;
  exportedSignals: {
    traces: boolean;
    metrics: boolean;
    logs: boolean;
  };
  fields: string;
  conditions: Condition[];
  destinationType: {
    type: string;
    displayName: string;
    imageUrl: string;
    supportedSignals: SupportedDestinationSignals;
  };
}

export const isActualDestination = (item: any): item is ActualDestination =>
  item && 'destinationType' in item;
