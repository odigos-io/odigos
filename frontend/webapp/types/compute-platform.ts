import { type PaginatedData } from './common';
import { type FetchedSource } from './sources';
import { type FetchedAction } from './actions';
import { type FetchedNamespace } from './namespace';
import { type InstrumentationRuleSpec } from './instrumentation-rules';
import { type TokenPayload, type Destination } from '@odigos/ui-containers';

export interface ComputePlatform {
  computePlatform: {
    computePlatformType?: string;
    apiTokens?: TokenPayload[];
    k8sActualNamespaces?: FetchedNamespace[];
    k8sActualNamespace?: FetchedNamespace;
    sources?: PaginatedData<FetchedSource>; // fetched is already "mapped", except for conditions (which are mapped by it's own component)
    destinations?: Destination[]; // fetched is already "mapped"
    actions?: FetchedAction[]; // should map from "FetchedAction" to "Action" in get-query
    instrumentationRules?: InstrumentationRuleSpec[]; // should map from "InstrumentationRuleSpec" to "InstrumentationRuleSpecMapped" in get-query
  };
}
