import { ActionType, BooleanOperation, JsonOperation, NumberOperation, StringOperation, type Condition } from '@odigos/ui-kit/types';

export interface FetchedAction {
  id: string;
  type: ActionType;
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
  collectReplicaSetAttributes?: boolean | null;
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
  services_name_filters?:
    | {
        service_name: string;
        sampling_ratio: number;
        fallback_sampling_ratio: number;
      }[]
    | null;
  attribute_filters?:
    | {
        service_name: string;
        attribute_key: string;
        fallback_sampling_ratio: number;
        condition: {
          string_condition?: {
            operation: StringOperation;
            expected_value?: string;
          };
          number_condition?: {
            operation: NumberOperation;
            expected_value?: number;
          };
          boolean_condition?: {
            operation: BooleanOperation;
            expected_value?: boolean;
          };
          json_condition?: {
            operation: JsonOperation;
            expected_value?: string;
            json_path?: string;
          };
        };
      }[]
    | null;
}

export interface ActionInput {
  type: ActionType;
  name: string;
  notes: string;
  disable: boolean;
  signals: string[]; // uppercase
  details: string;
}
