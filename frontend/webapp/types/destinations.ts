import type { Condition, DropdownOption, ExportedSignals } from './common';

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
  secret: boolean;
  initialValue: string;
  renderCondition: string[];
  hideFromReadData: string[];
  customReadDataLabels: {
    condition: string;
    title: string;
    value: string;
  }[];
}

export interface DynamicField {
  componentType: string;
  name: string;
  title: string;
  value: any;
  type?: string;
  placeholder?: string;
  required?: boolean;
  options?: DropdownOption[];
  renderCondition: string[];
}

export interface DestinationDetailsResponse {
  destinationTypeDetails: {
    fields: DestinationDetailsField[];
  };
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

export const isActualDestination = (item: any): item is ActualDestination => item && 'destinationType' in item;
