export enum DestinationsSortType {
  NAME = 'name',
  TYPE = 'type',
}

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
}

export interface Destination {
  name: string;
  id: string;
  fields: any;
  type: string;
  signals: {
    [key: string]: boolean;
  };
  destination_type: {
    image_url: string;
    display_name: string;
    supported_signals: {
      [key: string]: {
        supported: boolean;
      };
    };
    type: string;
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
