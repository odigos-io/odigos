import { NOTIFICATION_TYPE } from '@odigos/ui-utils';

export interface PaginatedData<T = any> {
  nextPage: string;
  items: T[];
}

export interface ExportedSignals {
  logs: boolean;
  metrics: boolean;
  traces: boolean;
}

export interface Notification {
  id: string;
  type: NOTIFICATION_TYPE;
  title?: string;
  message?: string;
  crdType?: string;
  target?: string;
  dismissed: boolean;
  seen: boolean;
  hideFromHistory?: boolean;
  time: string;
}
