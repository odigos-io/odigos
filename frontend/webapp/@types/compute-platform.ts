import type { FetchedSource } from './sources';
import type { FetchedAction } from './actions';
import type { FetchedNamespace } from './namespace';
import type { TokenPayload } from '@odigos/ui-utils';
import type { FetechedDestination } from './destinations';
import type { FetchedInstrumentationRule } from './instrumentation-rules';

interface PaginatedData<T = any> {
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
    destinations?: FetechedDestination[];
    actions?: FetchedAction[];
    instrumentationRules?: FetchedInstrumentationRule[];
  };
}
