import { ACTION_TYPE, type Condition } from '@odigos/ui-utils';

export interface FetchedAction {
  id: string;
  type: ACTION_TYPE;
  conditions: Condition[];
  spec: string;
}

// the stringified spec is parsed to this, which we still have to map to our ui-containers
export interface ParsedActionSpec {
  actionName: string;
  notes: string;
  signals: string[];
  disabled?: boolean;

  collectContainerAttributes?: boolean | null;
  collectWorkloadUID?: boolean | null;
  collectClusterUID?: boolean | null;
  labelsAttributes?: { labelKey: string; attributeKey: string }[] | null;
  annotationsAttributes?: { annotationKey: string; attributeKey: string }[] | null;
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
