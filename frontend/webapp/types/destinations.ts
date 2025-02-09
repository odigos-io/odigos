import type { ExportedSignals } from './common';
import type { DestinationCategories } from '@odigos/ui-utils';
import type { DestinationDynamicField, DestinationFormData } from '@odigos/ui-containers';

export interface FetchedDestinationCategories {
  destinationCategories: {
    categories: DestinationCategories;
  };
}

export interface FetchedDestinationDynamicField extends DestinationDynamicField {}

export interface DestinationInput extends DestinationFormData {}

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
