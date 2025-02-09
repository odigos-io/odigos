import { ACTION_TYPE, type FetchedCondition } from '@odigos/ui-utils';

export interface FetchedAction {
  id: string;
  type: ACTION_TYPE;
  conditions: FetchedCondition[];
  spec: string;
}

export interface ParsedActionSpec {
  actionName: string;
  notes: string;
  signals: string[];
  disabled?: boolean;

  clusterAttributes?: { attributeName: string; attributeStringValue: string }[] | null;
  attributeNamesToDelete?: string[] | null;
  renames?: { [oldKey: string]: string } | null;

  piiCategories?: string[] | null;
  fallback_sampling_ratio?: number | null;
  sampling_percentage?: string | null;
  endpoints_filters?:
    | {
        service_name: string;
        http_route: string;
        minimum_latency_threshold: number;
        fallback_sampling_ratio: number;
      }[]
    | null;
}

export interface ActionInput {
  type: ACTION_TYPE;
  name: string;
  notes: string;
  disable: boolean;
  signals: string[]; // uppercase
  details: string;
}
