export interface ExportedSignals {
  logs: boolean;
  metrics: boolean;
  traces: boolean;
}

export interface Condition {
  status: string;
  type: string;
  reason: string;
  message: string;
  lastTransitionTime: string;
}

export enum NOTIFICATION_TYPE {
  WARNING = 'warning',
  ERROR = 'error',
  SUCCESS = 'success',
  INFO = 'info',
  DEFAULT = 'default',
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

export type Config = {
  config: {
    installation: string;
  };
};

export interface DropdownOption {
  id: string;
  value: string;
}

export interface StepProps {
  title: string;
  subtitle?: string;
  state: 'finish' | 'active' | 'disabled';
  stepNumber: number;
}

export enum OVERVIEW_ENTITY_TYPES {
  RULE = 'rule',
  SOURCE = 'source',
  ACTION = 'action',
  DESTINATION = 'destination',
}

export enum OVERVIEW_NODE_TYPES {
  ADD_RULE = 'addRule',
  ADD_SOURCE = 'addSource',
  ADD_ACTION = 'addAction',
  ADD_DESTIONATION = 'addDestination',
}

export enum STATUSES {
  HEALTHY = 'healthy',
  UNHEALTHY = 'unhealthy',
}
