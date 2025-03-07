import type { Condition, DestinationCategories } from '@odigos/ui-utils';
import type { DestinationDynamicField, DestinationFormData } from '@odigos/ui-containers';

export interface FetchedDestination {
  id: string;
  name: string;
  exportedSignals: {
    traces: boolean;
    metrics: boolean;
    logs: boolean;
  };
  fields: string;
  conditions: Condition[] | null;
  destinationType: {
    type: string;
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
  };
}

export interface FetchedDestinationCategories {
  destinationCategories: {
    categories: DestinationCategories;
  };
}

export interface FetchedDestinationDynamicField extends DestinationDynamicField {}

export interface DestinationInput extends DestinationFormData {}
