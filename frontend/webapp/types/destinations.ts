import { Condition } from './common';

export enum DestinationsSortType {
  NAME = 'name',
  TYPE = 'type',
}

export interface DestinationTypeItem {
  displayName: string;
  imageUrl: string;
  category: 'managed' | 'self-hosted';
  type: string;
  testConnectionSupported: boolean;
  supportedSignals: {
    logs: {
      supported: boolean;
    };
    metrics: {
      supported: boolean;
    };
    traces: {
      supported: boolean;
    };
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

interface SupportedSignals {
  traces: SupportedSignal;
  metrics: SupportedSignal;
  logs: SupportedSignal;
}

export interface SelectedDestination {
  type: string;
  display_name: string;
  image_url: string;
  supported_signals: SupportedSignals;
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
    supported_signals: {
      traces: {
        supported: boolean;
      };
      metrics: {
        supported: boolean;
      };
      logs: {
        supported: boolean;
      };
    };
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
  signals: SupportedSignals;
  fields: {
    [key: string]: string;
  };
}
