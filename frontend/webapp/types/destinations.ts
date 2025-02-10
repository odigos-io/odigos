import type { DestinationCategories } from '@odigos/ui-utils';
import type { DestinationDynamicField, DestinationFormData } from '@odigos/ui-containers';

export interface FetchedDestinationCategories {
  destinationCategories: {
    categories: DestinationCategories;
  };
}

export interface FetchedDestinationDynamicField extends DestinationDynamicField {}

export interface DestinationInput extends DestinationFormData {}
