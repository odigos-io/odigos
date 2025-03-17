import type { FetchedSource } from './sources';
import type { FetchedAction } from './actions';
import type { FetchedNamespace } from './namespace';
import type { FetchedDestination } from './destinations';
import type { TokenPayload } from '@odigos/ui-kit/types';
import type { FetchedInstrumentationRule } from './instrumentation-rules';

export interface PaginatedData<T = any> {
  nextPage: string;
  items: T[];
}

export interface ComputePlatform {
  computePlatform: {
    computePlatformType?: string;
    apiTokens?: TokenPayload[];
    k8sActualNamespaces?: FetchedNamespace[];
    k8sActualNamespace?: FetchedNamespace;
    sources?: PaginatedData<FetchedSource>;
    source?: FetchedSource;
    destinations?: FetchedDestination[];
    actions?: FetchedAction[];
    instrumentationRules?: FetchedInstrumentationRule[];
  };
}
