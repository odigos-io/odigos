export interface ExportedSignals {
  logs: boolean;
  metrics: boolean;
  traces: boolean;
}

export interface Condition {
  type: string;
  status: string;
  message: string;
  lastTransitionTime: string;
}

export type NotificationType = 'warning' | 'error' | 'success' | 'info' | 'default';

export interface Notification {
  id: string;
  type: NotificationType;
  title?: string;
  message?: string;
  crdType?: string;
  target?: string;
  dismissed: boolean;
  seen: boolean;
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
