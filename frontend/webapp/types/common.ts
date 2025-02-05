import { NOTIFICATION_TYPE } from '@odigos/ui-utils';

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

export enum OVERVIEW_NODE_TYPES {
  ADD_RULE = 'addRule',
  ADD_SOURCE = 'addSource',
  ADD_ACTION = 'addAction',
  ADD_DESTINATION = 'addDestination',
}
