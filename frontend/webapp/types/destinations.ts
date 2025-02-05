import { type ExportedSignals } from './common';
import { type Comparison } from '@odigos/ui-utils';
import { type Destination } from '@odigos/ui-containers';
import { type DropdownProps } from '@odigos/ui-components';

type YamlCompareArr = [string, Comparison, string] | ['true' | 'false'];

export interface FetchedDestinationTypeItem {
  type: Destination['destinationType']['type'];
  displayName: Destination['destinationType']['displayName'];
  imageUrl: Destination['destinationType']['imageUrl'];
  supportedSignals: Destination['destinationType']['supportedSignals'];

  testConnectionSupported: boolean;
  fields: {
    [key: string]: string;
  };
}

export interface FetchedDestinationTypes {
  destinationTypes: {
    categories: {
      name: string;
      items: FetchedDestinationTypeItem[];
    }[];
  };
}

export interface FetchedDestinationDetailsField {
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

export interface FetchedDestinationDetailsResponse {
  destinationTypeDetails: {
    fields: FetchedDestinationDetailsField[];
  };
}

export interface FetchedDestinationDynamicField {
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
