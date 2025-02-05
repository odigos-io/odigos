import { type ExportedSignals } from './common';
import { type Comparison } from '@odigos/ui-utils';
import { type DropdownProps } from '@odigos/ui-components';

type YamlCompareArr = [string, Comparison, string] | ['true' | 'false'];

export interface DestinationTypeItem {
  type: string;
  testConnectionSupported: boolean;
  displayName: string;
  imageUrl: string;
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
  renderCondition: YamlCompareArr;
  hideFromReadData: YamlCompareArr;
  customReadDataLabels: {
    condition: string;
    title: string;
    value: string;
  }[];
}

export interface DestinationDetailsResponse {
  destinationTypeDetails: {
    fields: DestinationDetailsField[];
  };
}

export interface DynamicField {
  componentType: string;
  name: string;
  title: string;
  value: any;
  type?: string;
  placeholder?: string;
  required?: boolean;
  options?: DropdownProps['options'];
  renderCondition: YamlCompareArr;
}

export interface DestinationInput {
  type: string;
  name: string;
  exportedSignals: ExportedSignals;
  fields: {
    key: string;
    value: string;
  }[];
}

export interface ConfiguredDestination {
  type: string;
  displayName: string;
  imageUrl: string;
  category: string;
  exportedSignals: ExportedSignals;
  destinationTypeDetails: {
    title: string;
    value: string;
  }[];
}
